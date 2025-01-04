package controllers

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"github.com/LCGant/collaborativeeditor/models"
	"github.com/LCGant/collaborativeeditor/services"
)

type PageRequest struct {
	Subdomain string `json:"subdomain" binding:"required"`
}

type SaveRequest struct {
	Content        string `json:"content"`
	Subdomain      string `json:"subdomain" binding:"required"`
	ForceOverwrite bool   `json:"forceOverwrite"`
	UserKey        string `json:"userKey"`
}

type lastUpdateInfo struct {
	UpdatedAt time.Time
	UserKey   string
	Conn      *websocket.Conn
}

var (
	LastUpdateConn      = make(map[string]lastUpdateInfo)
	lastUpdateConnMutex = sync.RWMutex{}
)

var (
	clients      = make(map[*websocket.Conn]string)
	clientsMutex = sync.RWMutex{}
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// AccessOrCreatePageHandler Returns only “/editor/<subdomain>” to the front-end
func AccessOrCreatePageHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req PageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		baseURL := "/editor/" + req.Subdomain
		c.JSON(http.StatusOK, gin.H{"baseURL": baseURL})
	}
}

// GetPageHandler - creates (if it doesn't exist) and returns the HTML page
func GetPageHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		subdomainParam := c.Param("subdomain")
		contentParam := c.Param("content")
		if contentParam == "/" {
			contentParam = ""
		}
		subdomains := strings.TrimPrefix(subdomainParam+contentParam, "/")
		if subdomains == "" {
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<h1>No subdomain provided</h1>"))
			return
		}
		var page models.Page
		var parentID *uint
		subParts := strings.Split(subdomains, "/")
		for _, sub := range subParts {
			if sub == "" {
				continue
			}
			if err := services.EnsurePageExists(db, &page, sub, parentID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			parentID = &page.ID
		}
		userPage, err := services.GetPageFromURL(db, subdomains)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
			return
		}
		htmlContent := services.GenerateHTMLFromPage(userPage.Content)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
	}
}

// SavePageContentHandler Receives the content via JSON, saves it in the DB and handles conflicts
func SavePageContentHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SaveRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		page, err := services.GetPageFromURL(db, req.Subdomain)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
			return
		}
		lastUpdateConnMutex.RLock()
		lastInfo, exists := LastUpdateConn[req.Subdomain]
		lastUpdateConnMutex.RUnlock()
		if exists && !req.ForceOverwrite {
			if lastInfo.UserKey != req.UserKey {
				if time.Since(lastInfo.UpdatedAt) < 10*time.Second {
					c.JSON(http.StatusConflict, gin.H{
						"error":   "Content conflict detected",
						"content": page.Content,
					})
					return
				}
			}
		}
		newHTML := services.GenerateHTML2Save(req.Subdomain, req.Content)
		page.Content = newHTML
		page.UpdatedAt = time.Now()
		if err := db.Save(&page).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving content"})
			return
		}
		lastUpdateConnMutex.Lock()
		LastUpdateConn[req.Subdomain] = lastUpdateInfo{
			UpdatedAt: time.Now(),
			UserKey:   req.UserKey,
			Conn:      lastInfo.Conn,
		}
		lastUpdateConnMutex.Unlock()
		NotifyClients(req.Subdomain, req.Content)
		c.JSON(http.StatusOK, gin.H{"message": "Content saved successfully"})
	}
}

// GetPageContentHandler - returns the text of the <textarea> in JSON
func GetPageContentHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		subdomain := c.Param("subdomain")

		var page models.Page
		if err := db.Where("file_name = ?", subdomain).First(&page).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
			return
		}

		content := services.ExtractTextareaContent(page.Content)
		if content == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Textarea content not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"content":       content,
			"last_modified": page.UpdatedAt,
		})
	}
}

// GetChildPagesHandler - returns subpages of a given page
func GetChildPagesHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fullpath := strings.TrimPrefix(c.Query("fullpath"), "/")
		if fullpath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Full path not provided"})
			return
		}

		page, err := services.GetPageFromURL(db, fullpath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
			return
		}

		var childPages []models.Page
		if err := db.Where("parent_id = ?", page.ID).Find(&childPages).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching child pages"})
			return
		}

		var response []gin.H
		for _, child := range childPages {
			childFullPath := fullpath + "/" + child.FileName
			response = append(response, gin.H{
				"id":         child.ID,
				"file_name":  child.FileName,
				"full_path":  childFullPath,
				"level":      child.Level,
				"content":    child.Content,
				"updated_at": child.UpdatedAt,
			})
		}
		c.JSON(http.StatusOK, gin.H{"child_pages": response})
	}
}

// WebSocketHandler - route GET("/ws/:fullPath")
func WebSocketHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawParam := c.Param("fullPath")
		fullPath := strings.Trim(rawParam, "/")
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		clientsMutex.Lock()
		clients[conn] = fullPath
		clientsMutex.Unlock()
		for {
			_, msg, errMsg := conn.ReadMessage()
			if errMsg != nil {
				break
			}
			newContent := string(msg)
			notifyWsClients(fullPath, newContent)
		}
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()
	}
}

// NotifyClients  is called after saving (in SavePageContentHandler).
func NotifyClients(fullPath, content string) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()
	for conn, path := range clients {
		if path == fullPath {
			err := conn.WriteMessage(websocket.TextMessage, []byte(content))
			if err != nil {
				conn.Close()
				clientsMutex.RUnlock()
				clientsMutex.Lock()
				delete(clients, conn)
				clientsMutex.Unlock()
				clientsMutex.RLock()
				continue
			}
		}
	}
}

// notifyWsClients is called when a user types and sends via ws.send(...)
func notifyWsClients(fullPath, content string) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()
	for conn, path := range clients {
		if path == fullPath {
			err := conn.WriteMessage(websocket.TextMessage, []byte(content))
			if err != nil {
				conn.Close()
				clientsMutex.RUnlock()
				clientsMutex.Lock()
				delete(clients, conn)
				clientsMutex.Unlock()
				clientsMutex.RLock()
				continue
			}
		}
	}
}
