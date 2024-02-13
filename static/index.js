window.onload = function () {
    const entries = performance.getEntriesByType("navigation");
    if (entries.length !== 1) {
        console.warn("Expected 1 navigation event, got", entries);
        return;
    }
    const elapsed = Math.round(entries[0].domComplete) / 1000;
    document.getElementById("page-load-div").innerHTML = `<p>Loaded ${entries[0].name} in ${elapsed} seconds</p>`;
    document.getElementById("page-load-div").style.display = 'inline';
}
