document.addEventListener('DOMContentLoaded', function() {
    setInterval(function() {
        var blinks = document.getElementsByTagName('blink');
        for (var i = 0; i < blinks.length; i++) {
            blinks[i].style.visibility = blinks[i].style.visibility === 'hidden' ? 'visible' : 'hidden';
        }
    }, 500);
});
