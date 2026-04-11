// A classic 90s style mouse trail
(function() {
    var dots = [];
    var mouse = { x: 0, y: 0 };
    var colors = ["#ff00ff", "#00ffff", "#ffff00", "#00ff00", "#ff0000"];
    
    // Create elements
    for (var i = 0; i < 15; i++) {
        var n = document.createElement("div");
        n.innerHTML = "★";
        n.style.position = "absolute";
        n.style.pointerEvents = "none";
        n.style.zIndex = "9999";
        n.style.color = colors[i % colors.length];
        n.style.fontSize = (20 - i) + "px"; // get smaller at the end
        n.style.fontWeight = "bold";
        n.style.textShadow = "1px 1px #000";
        n.style.top = "-1000px"; // hide initially
        n.style.left = "-1000px";
        document.body.appendChild(n);
        dots.push({ x: 0, y: 0, node: n });
    }

    document.addEventListener("mousemove", function(event) {
        mouse.x = event.pageX + 5;
        mouse.y = event.pageY + 5;
    });

    function draw() {
        var x = mouse.x;
        var y = mouse.y;

        dots.forEach(function(dot, index, arr) {
            var nextDot = arr[index + 1] || arr[0];
            dot.x = x;
            dot.y = y;
            dot.node.style.left = dot.x + "px";
            dot.node.style.top = dot.y + "px";
            
            // The trail effect math
            x += (nextDot.x - dot.x) * .6;
            y += (nextDot.y - dot.y) * .6;
        });
    }

    function animate() {
        draw();
        requestAnimationFrame(animate);
    }
    
    // Start loop
    animate();
})();