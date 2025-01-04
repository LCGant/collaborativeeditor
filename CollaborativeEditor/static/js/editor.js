document.addEventListener("DOMContentLoaded", () => {
  const editor = document.getElementById("editor");
  if (!editor) {
    console.warn("Editor textarea not found.");
    return;
  }

  let userKey = localStorage.getItem("userKey");
  if (!userKey) {
    if (crypto?.randomUUID) {
      userKey = crypto.randomUUID();
    } else {
      userKey = "user_" + Math.random().toString(36).substring(2, 8);
    }
    localStorage.setItem("userKey", userKey);
  }
  console.log("[DEBUG] userKey:", userKey);

  let isTyping = false;
  let saveTimeout = null;
  const DEBOUNCE_MS = 5000;

  editor.addEventListener("input", () => {
    isTyping = true;
    clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
      isTyping = false; 
      saveContent(editor.value);
    }, DEBOUNCE_MS);
  });

  async function saveContent(content, forceOverwrite = false) {
    const fullPath = getCurrentSubdomain();
    const payload = {
      content,
      subdomain: fullPath,
      forceOverwrite,
      userKey
    };

    try {
      const response = await fetch("/save_page_content", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Accept": "application/json"
        },
        body: JSON.stringify(payload)
      });

      if (response.ok) {
        showNotification("Content saved!");
        loadChildPages();
      } else if (response.status === 409) {
        const data = await response.json();
        console.warn("Conflict detected:", data);
        showConflictModal(content, data.content);
      } else {
        console.error("Save error. Status:", response.status);
      }
    } catch (err) {
      console.error("Error saving content:", err);
    }
  }

  function showConflictModal(localContent, serverContent) {
    let modal = document.getElementById("conflict-modal");
    if (modal) return;

    modal = document.createElement("div");
    modal.id = "conflict-modal";
    Object.assign(modal.style, {
      position: "fixed",
      top: "0",
      left: "0",
      width: "100%",
      height: "100%",
      backgroundColor: "rgba(0,0,0,0.5)",
      display: "flex",
      justifyContent: "center",
      alignItems: "center",
      zIndex: 9999
    });

    const modalContent = document.createElement("div");
    Object.assign(modalContent.style, {
      backgroundColor: "#fff",
      padding: "20px",
      borderRadius: "5px",
      maxWidth: "400px",
      textAlign: "center",
      fontFamily: "Arial, sans-serif"
    });

    const message = document.createElement("p");
    message.textContent = "Outro usuário atualizou o texto. O que você deseja fazer?";
    modalContent.appendChild(message);

    const buttonsContainer = document.createElement("div");
    Object.assign(buttonsContainer.style, {
      display: "flex",
      justifyContent: "space-around",
      marginTop: "20px"
    });

    const keepMineBtn = document.createElement("button");
    keepMineBtn.textContent = "Manter meu texto";
    Object.assign(keepMineBtn.style, {
      backgroundColor: "#3498db",
      color: "#fff",
      border: "none",
      borderRadius: "3px",
      padding: "10px 15px",
      cursor: "pointer"
    });
    keepMineBtn.addEventListener("click", async () => {
      document.body.removeChild(modal);
      await saveContent(localContent, true);
    });
    buttonsContainer.appendChild(keepMineBtn);

    const discardMineBtn = document.createElement("button");
    discardMineBtn.textContent = "Descartar meu texto";
    Object.assign(discardMineBtn.style, {
      backgroundColor: "#e74c3c",
      color: "#fff",
      border: "none",
      borderRadius: "3px",
      padding: "10px 15px",
      cursor: "pointer"
    });
    discardMineBtn.addEventListener("click", async () => {
      document.body.removeChild(modal);
      await fetchServerContent();
    });
    buttonsContainer.appendChild(discardMineBtn);

    modalContent.appendChild(buttonsContainer);
    modal.appendChild(modalContent);
    document.body.appendChild(modal);
  }

  async function fetchServerContent() {
    const fullPath = getCurrentSubdomain();
    try {
      const response = await fetch(`/get_page_content/${fullPath}`, {
        method: "GET",
        headers: { "Accept": "application/json" }
      });
      if (response.ok) {
        const data = await response.json();
        editor.value = data.content || "";
        showNotification("Content updated from server.");
      } else {
        console.error("Error fetching server content. Status:", response.status);
      }
    } catch (err) {
      console.error("Error fetching server content:", err);
    }
  }

  const enableWebSocket = true;
  if (enableWebSocket) {
    const protocol = (window.location.protocol === "https:") ? "wss" : "ws";
    const ws = new WebSocket(`${protocol}://${window.location.host}/ws/${getCurrentSubdomain()}`);

    ws.onopen = () => console.log("[DEBUG] WebSocket connected.");

    ws.onmessage = (event) => {
      const newContent = event.data;
      if (isTyping) {
        if (confirm("Outro usuário salvou alterações. Deseja aplicar agora?")) {
          editor.value = newContent;
          isTyping = false;
        } else {
          console.warn("Atualização do servidor ignorada temporariamente.");
        }
      } else {
        editor.value = newContent;
        console.log("Editor updated automatically from server.");
      }
    };

    ws.onclose = () => {
      console.warn("WebSocket closed.");
    };
    ws.onerror = (err) => {
      console.error("WebSocket error:", err);
    };
  }

  createSidebar();
  loadChildPages();

  function createSidebar() {
    const sidebar = document.createElement("div");
    sidebar.id = "sidebar";
    Object.assign(sidebar.style, {
      width: "250px",
      backgroundColor: "#2c3e50",
      color: "#ecf0f1",
      padding: "15px",
      boxSizing: "border-box",
      overflowY: "auto",
      position: "fixed",
      left: "0",
      top: "0",
      height: "100%",
      fontFamily: "Arial, sans-serif",
      zIndex: "999",
      transition: "all 0.3s ease",
      borderRadius: "0"
    });

    const title = document.createElement("h3");
    title.textContent = "Child Pages";
    Object.assign(title.style, {
      marginTop: "0",
      marginBottom: "15px",
      fontSize: "18px",
      borderBottom: "1px solid #34495e",
      paddingBottom: "10px"
    });
    sidebar.appendChild(title);

    const list = document.createElement("ul");
    list.id = "child-pages-list";
    Object.assign(list.style, {
      listStyleType: "none",
      padding: "0",
      margin: "0"
    });
    sidebar.appendChild(list);

    document.body.appendChild(sidebar);
    const editorContainer = document.createElement("div");
    editorContainer.id = "editor-container";
    Object.assign(editorContainer.style, {
      marginLeft: "250px",
      height: "100%",
      display: "flex",
      flexDirection: "column",
      width: "calc(100% - 250px)"
    });

    if (editor) {
      const parent = editor.parentNode;
      parent.removeChild(editor);
      Object.assign(editor.style, {
        height: "100%",
        width: "100%"
      });
      editorContainer.appendChild(editor);
    }

    document.body.appendChild(editorContainer);
  }

  async function loadChildPages() {
    const list = document.getElementById("child-pages-list");
    if (!list) return;
    list.innerHTML = "";

    const currentPath = getCurrentSubdomain();
    try {
      const response = await fetch(`/getchildreneditor?fullpath=${encodeURIComponent(currentPath)}`, {
        method: "GET",
        headers: { "Accept": "application/json" }
      });
      if (response.ok) {
        const data = await response.json();
        const childPages = Array.isArray(data.child_pages) ? data.child_pages : [];
        if (childPages.length === 0) {
          const noPagesItem = document.createElement("li");
          noPagesItem.textContent = "No child pages.";
          Object.assign(noPagesItem.style, {
            padding: "8px",
            fontStyle: "italic"
          });
          list.appendChild(noPagesItem);
          return;
        }

        childPages.forEach(page => {
          const li = document.createElement("li");
          li.textContent = page.file_name;
          Object.assign(li.style, {
            padding: "8px",
            cursor: "pointer",
            borderBottom: "1px solid #34495e"
          });

          li.addEventListener("click", () => {
            window.location.href = `/editor/${page.full_path}`;
          });

          li.addEventListener("mouseover", () => {
            li.style.backgroundColor = "#34495e";
          });
          li.addEventListener("mouseout", () => {
            li.style.backgroundColor = "transparent";
          });

          list.appendChild(li);
        });
      } else {
        console.error("Error fetching child pages.");
        const errLi = document.createElement("li");
        errLi.textContent = "Error loading child pages.";
        Object.assign(errLi.style, {
          padding: "8px",
          color: "#e74c3c",
          fontStyle: "italic"
        });
        list.appendChild(errLi);
      }
    } catch (err) {
      console.error("Error fetching child pages:", err);
      const errLi = document.createElement("li");
      errLi.textContent = "Error loading child pages.";
      Object.assign(errLi.style, {
        padding: "8px",
        color: "#e74c3c",
        fontStyle: "italic"
      });
      list.appendChild(errLi);
    }
  }

  function getCurrentSubdomain() {
    return window.location.pathname
      .replace(/^\/editor\//, "")
      .replace(/\/$/, "");
  }

  function showNotification(msg, bgColor = "#2ecc71") {
    const div = document.createElement("div");
    div.textContent = msg;
    Object.assign(div.style, {
      position: "fixed",
      top: "20px",
      right: "20px",
      padding: "10px 20px",
      borderRadius: "5px",
      backgroundColor: bgColor,
      color: "#fff",
      zIndex: 1000,
      opacity: 0,
      transition: "opacity 0.5s"
    });
    document.body.appendChild(div);
    setTimeout(() => (div.style.opacity = 1), 10);
    setTimeout(() => {
      div.style.opacity = 0;
      setTimeout(() => {
        document.body.removeChild(div);
      }, 500);
    }, 2000);
  }

});
