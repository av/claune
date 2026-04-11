// Wait for Webamp to load
const checkWebamp = setInterval(() => {
    if (window.Webamp) {
        clearInterval(checkWebamp);
        initWebamp();
    }
}, 100);

function initWebamp() {
    if(!Webamp.browserIsSupported()) {
        console.error("Webamp not supported");
        return;
    }
    const webamp = new Webamp({
        initialTracks: [
            {
                metaData: {
                    artist: "Unknown MIDI Artist",
                    title: "Geocities Anthem (bg.mp3)"
                },
                url: "assets/sounds/bg.mp3",
                duration: 10
            }
        ],
    });
    webamp.renderWhenReady(document.getElementById('winamp-container'));
}
