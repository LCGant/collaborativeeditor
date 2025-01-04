// static/js/index.js

document.addEventListener("DOMContentLoaded", () => {
    const subdomainInput = document.getElementById("subdomain");
    const goToEditorButton = document.getElementById("goToEditor");
  
    goToEditorButton.addEventListener("click", async () => {
      const subdomain = subdomainInput.value.trim();
      if (!subdomain) {
        showNotification("Please enter a subdomain.");
        return;
      }
  
      try {
        const response = await fetch("/access_or_create_page", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "Accept": "application/json"
          },
          body: JSON.stringify({ subdomain }),
          credentials: "include",
        });
  
        if (!response.ok) {
          const errorData = await response.json();
          showNotification(errorData.error || "An error occurred while accessing the page.");
          return;
        }
  
        const data = await response.json();
        // Redireciona para a página do editor
        window.location.href = data.baseURL; 
  
      } catch (error) {
        showNotification("An unexpected error occurred.");
        console.error(error);
      }
    });
  });
  
  function toggleNotification() {
    const notificationBalloon = document.getElementById('notificationBalloon');
    const notificationIcon = document.getElementById('notificationIcon');
    const isDisplayed = notificationBalloon.style.display === 'flex';
    notificationBalloon.style.display = isDisplayed ? 'none' : 'flex';
    notificationIcon.style.display = isDisplayed ? 'none' : 'flex';
  }
  
  function closeNotification() {
    const notificationBalloon = document.getElementById('notificationBalloon');
    const notificationIcon = document.getElementById('notificationIcon');
    notificationBalloon.style.display = 'none';
    notificationIcon.style.display = 'none';
  }
  
  function showNotification(message, timeout = 3000) {
    document.getElementById('notificationMessage').innerText = message;
    const notificationBalloon = document.getElementById('notificationBalloon');
    const notificationIcon = document.getElementById('notificationIcon');
    
    notificationBalloon.style.display = 'flex';
    notificationBalloon.style.flexDirection = 'column';
    notificationBalloon.style.gap = '10px';
    
    notificationIcon.style.display = 'flex';
    notificationIcon.style.flexDirection = 'column';
    notificationIcon.style.gap = '10px';
  
    const balloonHeight = notificationBalloon.offsetHeight;
    notificationIcon.style.top = `${20 + balloonHeight + 10}px`;
  
    setTimeout(closeNotification, timeout);
  }
  