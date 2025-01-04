package services

import (
	"errors"
	"html"
	"regexp"
	"strings"

	"github.com/LCGant/collaborativeeditor/models"
	"gorm.io/gorm"
)

// GenerateDefaultHTML returns the initial HTML for a new page
func GenerateDefaultHTML(subdomain string) string {
	return `<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>` + subdomain + ` - Editor</title>
		<style>
			body, html {
				margin: 0;
				padding: 0;
				height: 100%;
				width: 100%;
				display: flex;
				justify-content: center;
				align-items: center;
				background-color: #f9f9f9;
			}
			textarea {
				width: 100%;
				height: 100%;
				padding: 10px;
				border: none;
				resize: none;
				outline: none;
				font-family: Arial, sans-serif;
				font-size: 16px;
				line-height: 1.5;
				box-sizing: border-box;
				overflow: auto;
			}
			textarea:focus {
				outline: none;
			}
		</style>
	</head>
	<body>
		<textarea id="editor" placeholder="Write here..."></textarea>
		<script src="/static/js/editor.js"></script>
	</body>
	</html>`
}

// GenerateHTML2Save escapes the user's content and injects it into <textarea>
func GenerateHTML2Save(subdomain, content string) string {
	escapedContent := html.EscapeString(content)
	return `<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>` + subdomain + ` - Editor</title>
		<style>
			body, html {
				margin: 0;
				padding: 0;
				height: 100%;
				width: 100%;
				display: flex;
				justify-content: center;
				align-items: center;
				background-color: #f9f9f9;
			}
			textarea {
				width: 100%;
				height: 100%;
				padding: 10px;
				border: none;
				resize: none;
				outline: none;
				font-family: Arial, sans-serif;
				font-size: 16px;
				line-height: 1.5;
				box-sizing: border-box;
				overflow: auto;
			}
			textarea:focus {
				outline: none;
			}
		</style>
	</head>
	<body>
		<textarea id="editor" placeholder="Write here...">` + escapedContent + `</textarea>
		<script src="/static/js/editor.js"></script>
	</body>
	</html>`
}

// GenerateHTMLFromPage simply returns the stored HTML
func GenerateHTMLFromPage(content string) string {
	return content
}

// ExtractTextareaContent extracts only what is inside <textarea>...</textarea>
func ExtractTextareaContent(htmlContent string) string {
	re := regexp.MustCompile(`(?s)<textarea.*?>(.*?)</textarea>`)
	matches := re.FindStringSubmatch(htmlContent)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// EnsurePageExists checks if (sub, parentID) exists; if not, creates
func EnsurePageExists(db *gorm.DB, page *models.Page, sub string, parentID *uint) error {
	if sub == "" {
		return nil
	}
	query := db.Where("file_name = ?", sub)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", parentID)
	}

	var existingPage models.Page
	err := query.First(&existingPage).Error
	if err == nil {
		*page = existingPage
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return errors.New("error checking page existence")
	}

	newPage := models.Page{
		FileName: sub,
		ParentID: parentID,
		Content:  GenerateDefaultHTML(sub),
	}

	newPage.Level = 0
	if parentID != nil {
		var parentPage models.Page
		if err := db.First(&parentPage, *parentID).Error; err == nil {
			newPage.Level = parentPage.Level + 1
		}
	}

	if err := rulesEnsurePageExists(db, &newPage); err != nil {
		return err
	}

	if err := db.Create(&newPage).Error; err != nil {
		return errors.New("error creating page in database")
	}

	*page = newPage
	return nil
}

// rulesEnsurePageExists imposes rules (e.g. max 3 levels, max 5 child pages)
func rulesEnsurePageExists(db *gorm.DB, newPage *models.Page) error {
	if newPage.Level >= 3 {
		return errors.New("maximum page level reached")
	}
	if newPage.ParentID != nil {
		var count int64
		if err := db.Model(&models.Page{}).
			Where("parent_id = ?", newPage.ParentID).
			Count(&count).Error; err == nil {
			if count >= 5 {
				return errors.New("maximum number of child pages reached")
			}
		}
	}
	return nil
}

// GetPageFromURL searches for a page by its URL and returns it.
func GetPageFromURL(db *gorm.DB, url string) (*models.Page, error) {
	subdomainParts := strings.Split(url, "/")
	var parentID *uint
	level := 0
	var page models.Page
	for _, sub := range subdomainParts {
		page = models.Page{}
		query := db.Where("file_name = ?", sub)
		if parentID == nil {
			query = query.Where("parent_id IS NULL")
		} else {
			query = query.Where("parent_id = ?", parentID)
		}
		query = query.Where("level = ?", level)
		err := query.First(&page).Error
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("page not found")
		} else if err != nil {
			return nil, errors.New("error querying page")
		}
		copiedID := page.ID
		parentID = &copiedID
		level++
	}
	return &page, nil
}
