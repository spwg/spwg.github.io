const NUMLEVELS = 5;
const PEG_NOT_SELECTED = 0;
const PEG_SELECTED_FROM = 1;
const PEG_SELECTED_TO = 2;
const PEG_STATE = {
    0: PEG_NOT_SELECTED,
    1: PEG_NOT_SELECTED,
    2: PEG_NOT_SELECTED,
    3: PEG_NOT_SELECTED,
    4: PEG_NOT_SELECTED,
    5: PEG_NOT_SELECTED,
    6: PEG_NOT_SELECTED,
    7: PEG_NOT_SELECTED,
    8: PEG_NOT_SELECTED,
    9: PEG_NOT_SELECTED,
    10: PEG_NOT_SELECTED,
    11: PEG_NOT_SELECTED,
    12: PEG_NOT_SELECTED,
    13: PEG_NOT_SELECTED,
    14: PEG_NOT_SELECTED,
    15: PEG_NOT_SELECTED,
}

function assert(cond, msg) {
    if (!cond) {
        if (msg) console.error(msg);
        throw new Error("assertion failed");
    }
}

function calcCoords(canvas, row, hole) {
    /* The hole in the top row is halfway between the left and
    right sides of the canvas, and it's 20 pixels from the top.
    Each new row starts at the leftmost hole 20 pixels to the leftmost
    hole in the row above it. So, if row N starts at X, then row 
    N+1 starts at X-20. */
    
    assert(row >= 0 && row <= NUMLEVELS);
    assert(hole >= 0 && hole <= row);
    
    const SPACING = canvas.height / NUMLEVELS - 10;
    const leftmostHole = canvas.width / 2 - SPACING * row / 2;
    const x = leftmostHole + SPACING * hole;
    const y = SPACING * (row + 1);
    return [x, y];
}

function drawBoard(canvas, ctx) {
    for (let row = 0; row < NUMLEVELS; row++) {
        const nHoles = row + 1;
        for (let hole = 0; hole < nHoles; hole++) {
            const [x, y] = calcCoords(canvas, row, hole);
            ctx.beginPath();
            ctx.arc(x, y, 10, 0, 2 * Math.PI);
            ctx.stroke();
        }
    }
}

function handleClick(event) {
    const x = event.clientX;
    const y = event.clientY;
    const node = event.target;
    const canvas = document.getElementById('canvas');
    const ctx = canvas.getContext('2d');
    
}

function newGame() {
    console.log("starting a new game")
    
    const canvas = document.getElementById("canvas");
    const ctx = canvas.getContext("2d");
    
    console.log("canvas", canvas);
    console.log("ctx", ctx);
    
    console.log("drawing the board");
    drawBoard(canvas, ctx);

    console.log("listening for click events");
    canvas.addEventListener("click", handleClick);
    
}

console.log("running iqtestgame.js");
document.addEventListener("DOMContentLoaded", newGame);
