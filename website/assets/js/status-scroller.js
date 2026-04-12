var statusText = " ~*~ WELCOME TO CLAUNE !! THE ULTIMATE MEME SOUNDBOARD FOR CLAUDE CODE !! ~*~ xXx_DOWNLOAD_TODAY_xXx ~*~ ";
var statusSpeed = 150;
var statusPos = 0;

function scrollStatus() {
    window.status = statusText.substring(statusPos, statusText.length) + statusText.substring(0, statusPos);
    statusPos++;
    if (statusPos > statusText.length) {
        statusPos = 0;
    }
    setTimeout(scrollStatus, statusSpeed);
}

// Attach gracefully to onload
var oldStatusOnLoad = window.onload;
window.onload = function() {
    scrollStatus();
    if (typeof oldStatusOnLoad === 'function') {
        oldStatusOnLoad();
    }
};