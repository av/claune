var snowmax = 35;
var snowcolors = ["#FF00FF", "#00FFFF", "#FFFF00", "#FFFFFF", "#00FF00"];
var snowtype = ["Comic Sans MS", "Arial", "Times New Roman"];
var snowletters = ["*", "🎵", "♪", "❄", "💀"];
var sinkspeed = 0.8;
var snowmaxsize = 28;
var snowminsize = 12;

var snow = [];
var marginbottom;
var marginright;
var x_mv = [];
var crds = [];
var lftrght = [];

function randommaker(range) {
    return Math.floor(range * Math.random());
}

function initsnow() {
    marginbottom = window.innerHeight || document.body.clientHeight || document.documentElement.clientHeight;
    marginright = window.innerWidth || document.body.clientWidth || document.documentElement.clientWidth;

    var snowsizerange = snowmaxsize - snowminsize;
    for (var i = 0; i <= snowmax; i++) {
        var span = document.createElement("span");
        span.id = "s" + i;
        span.style.position = "absolute";
        span.style.top = "-" + snowmaxsize + "px";
        span.style.zIndex = "999";
        span.style.pointerEvents = "none";
        
        var letter = snowletters[randommaker(snowletters.length)];
        span.innerHTML = letter;
        document.body.appendChild(span);

        crds[i] = 0;
        lftrght[i] = Math.random() * 15;
        x_mv[i] = 0.03 + Math.random() / 10;
        snow[i] = span;
        
        snow[i].size = randommaker(snowsizerange) + snowminsize;
        snow[i].style.fontSize = snow[i].size + "px";
        snow[i].style.color = snowcolors[randommaker(snowcolors.length)];
        snow[i].style.fontFamily = snowtype[randommaker(snowtype.length)];
        snow[i].sink = sinkspeed * snow[i].size / 5;
        snow[i].posx = randommaker(marginright - snow[i].size);
        snow[i].posy = randommaker(marginbottom - 2 * snow[i].size);
        snow[i].style.left = snow[i].posx + "px";
        snow[i].style.top = snow[i].posy + "px";
    }
    movesnow();
}

function movesnow() {
    marginbottom = window.innerHeight || document.body.clientHeight || document.documentElement.clientHeight;
    marginright = window.innerWidth || document.body.clientWidth || document.documentElement.clientWidth;
    var scrollY = window.scrollY || document.documentElement.scrollTop || document.body.scrollTop || 0;

    for (var i = 0; i <= snowmax; i++) {
        crds[i] += x_mv[i];
        snow[i].posy += snow[i].sink;
        snow[i].style.left = snow[i].posx + lftrght[i] * Math.sin(crds[i]) + "px";
        snow[i].style.top = snow[i].posy + scrollY + "px";

        if (snow[i].posy >= marginbottom - 2 * snow[i].size || parseInt(snow[i].style.left) > (marginright - 3 * lftrght[i])) {
            snow[i].posx = randommaker(marginright - snow[i].size);
            snow[i].posy = 0;
        }
    }
    setTimeout(movesnow, 50);
}

if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initsnow);
} else {
    initsnow();
}
