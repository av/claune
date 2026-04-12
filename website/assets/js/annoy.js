document.addEventListener('click', function(e) {
  // Check if the clicked target or any of its parents is an anchor tag
  let el = e.target;
  while (el && el.tagName !== 'A' && el !== document.body) {
    el = el.parentElement;
  }
  
  // If it's not a link, 20% chance to annoy
  if (!el || el.tagName !== 'A') {
    if (Math.random() < 0.20) {
      if (Math.random() < 0.5) {
        alert("DON'T CLICK THERE!!");
      } else {
        alert("ERROR 404: MOUSE TOO FAST");
      }
    }
  }
});
