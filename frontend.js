let board, game = new Chess(), statusEl, fenEl, pgnEl;

const onDragStart = (source, piece, position, orientation) => {
    // will prevent pieces from dragging that aren't allowed to
    const gameOver = game.game_over();
    const whitesTurn = game.turn() === "w";
    const blacksTurn = game.turn() === "b";
    const whiteMoving = piece.search("/^b/") !== -1;
    const blackMoving = piece.search("/^w/") !== -1;
    const whitesTurnButBlackMoving = whitesTurn && blackMoving;
    const blacksTurnButWhiteMoving = blacksTurn && whiteMoving;
    return !(gameOver || whitesTurnButBlackMoving || blacksTurnButWhiteMoving);
}

const onDrop = (source, target) => {
    // will snapback the move if it's not allowed
    const move = game.move({
      from: source,
      to: target,
      promotion: 'q' // pawns always promoted to queens
    });
    if (move === null) {
      return "snapback";
    }
    updateStatus();
}

const playMoveForBlack = (game) => {
    // given a game variable, will make a move
    // if it is black's turn
    if (game.turn() === "b") {
	      mmDriver();
    }
}

const reverseArray = function(array) {
    return array.slice().reverse();
};

// data for evaluation function from chessprogramming.com

const pawnEvalWhite = [
    [0.0,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0],
    [5.0,  5.0,  5.0,  5.0,  5.0,  5.0,  5.0,  5.0],
    [1.0,  1.0,  2.0,  3.0,  3.0,  2.0,  1.0,  1.0],
    [0.5,  0.5,  1.0,  2.5,  2.5,  1.0,  0.5,  0.5],
    [0.0,  0.0,  0.0,  2.0,  2.0,  0.0,  0.0,  0.0],
    [0.5, -0.5, -1.0,  0.0,  0.0, -1.0, -0.5,  0.5],
    [0.5,  1.0, 1.0,  -2.0, -2.0,  1.0,  1.0,  0.5],
    [0.0,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0]];
const pawnEvalBlack = reverseArray(pawnEvalWhite);
const knightEval = [
    [-5.0, -4.0, -3.0, -3.0, -3.0, -3.0, -4.0, -5.0],
    [-4.0, -2.0,  0.0,  0.0,  0.0,  0.0, -2.0, -4.0],
    [-3.0,  0.0,  1.0,  1.5,  1.5,  1.0,  0.0, -3.0],
    [-3.0,  0.5,  1.5,  2.0,  2.0,  1.5,  0.5, -3.0],
    [-3.0,  0.0,  1.5,  2.0,  2.0,  1.5,  0.0, -3.0],
    [-3.0,  0.5,  1.0,  1.5,  1.5,  1.0,  0.5, -3.0],
    [-4.0, -2.0,  0.0,  0.5,  0.5,  0.0, -2.0, -4.0],
    [-5.0, -4.0, -3.0, -3.0, -3.0, -3.0, -4.0, -5.0]];
const bishopEvalWhite = [
    [ -2.0, -1.0, -1.0, -1.0, -1.0, -1.0, -1.0, -2.0],
    [ -1.0,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0, -1.0],
    [ -1.0,  0.0,  0.5,  1.0,  1.0,  0.5,  0.0, -1.0],
    [ -1.0,  0.5,  0.5,  1.0,  1.0,  0.5,  0.5, -1.0],
    [ -1.0,  0.0,  1.0,  1.0,  1.0,  1.0,  0.0, -1.0],
    [ -1.0,  1.0,  1.0,  1.0,  1.0,  1.0,  1.0, -1.0],
    [ -1.0,  0.5,  0.0,  0.0,  0.0,  0.0,  0.5, -1.0],
    [ -2.0, -1.0, -1.0, -1.0, -1.0, -1.0, -1.0, -2.0]];
const bishopEvalBlack = reverseArray(bishopEvalWhite);
const rookEvalWhite = [
    [  0.0,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0],
    [  0.5,  1.0,  1.0,  1.0,  1.0,  1.0,  1.0,  0.5],
    [ -0.5,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0, -0.5],
    [ -0.5,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0, -0.5],
    [ -0.5,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0, -0.5],
    [ -0.5,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0, -0.5],
    [ -0.5,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0, -0.5],
    [  0.0,   0.0, 0.0,  0.5,  0.5,  0.0,  0.0,  0.0]];
const rookEvalBlack = reverseArray(rookEvalWhite);
const evalQueen =[
    [ -2.0, -1.0, -1.0, -0.5, -0.5, -1.0, -1.0, -2.0],
    [ -1.0,  0.0,  0.0,  0.0,  0.0,  0.0,  0.0, -1.0],
    [ -1.0,  0.0,  0.5,  0.5,  0.5,  0.5,  0.0, -1.0],
    [ -0.5,  0.0,  0.5,  0.5,  0.5,  0.5,  0.0, -0.5],
    [  0.0,  0.0,  0.5,  0.5,  0.5,  0.5,  0.0, -0.5],
    [ -1.0,  0.5,  0.5,  0.5,  0.5,  0.5,  0.0, -1.0],
    [ -1.0,  0.0,  0.5,  0.0,  0.0,  0.0,  0.0, -1.0],
    [ -2.0, -1.0, -1.0, -0.5, -0.5, -1.0, -1.0, -2.0]];
const kingEvalWhite = [
    [ -3.0, -4.0, -4.0, -5.0, -5.0, -4.0, -4.0, -3.0],
    [ -3.0, -4.0, -4.0, -5.0, -5.0, -4.0, -4.0, -3.0],
    [ -3.0, -4.0, -4.0, -5.0, -5.0, -4.0, -4.0, -3.0],
    [ -3.0, -4.0, -4.0, -5.0, -5.0, -4.0, -4.0, -3.0],
    [ -2.0, -3.0, -3.0, -4.0, -4.0, -3.0, -3.0, -2.0],
    [ -1.0, -2.0, -2.0, -2.0, -2.0, -2.0, -2.0, -1.0],
    [  2.0,  2.0,  0.0,  0.0,  0.0,  0.0,  2.0,  2.0 ],
    [  2.0,  3.0,  1.0,  0.0,  0.0,  1.0,  3.0,  2.0 ]];
const kingEvalBlack = reverseArray(kingEvalWhite);

// end evaluation function data

// the AI part
const getPieceValue = function (piece, x, y) {
    if (piece === null) {
        return 0;
    }
    const getAbsoluteValue = function (piece, isWhite, x ,y) {
        if (piece.type === 'p') {
            return 10 + (isWhite ? pawnEvalWhite[y][x] : pawnEvalBlack[y][x] );
        } else if (piece.type === 'r') {
            return 50 + (isWhite ? rookEvalWhite[y][x] : rookEvalBlack[y][x] );
        } else if (piece.type === 'n') {
            return 30 + knightEval[y][x];
        } else if (piece.type === 'b') {
            return 30 + (isWhite ? bishopEvalWhite[y][x] : bishopEvalBlack[y][x] );
        } else if (piece.type === 'q') {
            return 90 + evalQueen[y][x];
        } else if (piece.type === 'k') {
            return 900 + (isWhite ? kingEvalWhite[y][x] : kingEvalBlack[y][x] );
        }
        throw "Unknown piece type: " + piece.type;
    };
    const absoluteValue = getAbsoluteValue(piece, piece.color === 'w', x ,y);
    return piece.color === 'w' ? absoluteValue : -absoluteValue;
};

const mmDriver = () => {
    const now = Date.now();
    const searchDepth = 3;
    const moves = game.moves();
    const alpha = -10000, beta = 10000;
    let bestMove = null, bestScore = Number.NEGATIVE_INFINITY;
    for (let i = 0; i < moves.length; i++) {
      const move = moves[i];
	    game.move(move);
	    const score = miniMax(searchDepth - 1, game, alpha, beta, false);
      game.undo();
	    if (score > bestScore) {
	      bestScore = score;
	      bestMove = move;
	    }
    }
    console.log(Date.now() - now);
    game.move(bestMove);
    updateStatus();
    board.position(game.fen());
}

const miniMax = (depth, game, alpha, beta, isMaximizing) => {
    // will perform the minimax algorithm with alpha-beta pruning
    if (depth === 0) {
	    const base = -evalBoard(game);
	    return base;
    }
    const moves = game.moves();
    if (isMaximizing) {
	     let bestMove = Number.NEGATIVE_INFINITY;
	     for (let i = 0; i < moves.length; i++) {
	        game.move(moves[i]);
	        const recCall = miniMax(depth - 1, game, alpha, beta, false);
	        game.undo();
          bestMove = Math.max(bestMove, recCall);
          alpha = Math.max(alpha, bestMove);
          if (beta <= alpha) {
            break;
          }
      }
      return bestMove;
    } else {
      let bestMove = Number.POSITIVE_INFINITY;
      for (let i = 0; i < moves.length; i++) {
        game.move(moves[i]);
        const recCall = miniMax(depth - 1, game, alpha, beta, true);
        game.undo();
        bestMove = Math.min(bestMove, recCall);
        beta = Math.min(beta, bestMove);
        if (beta <= alpha) {
          break;
        }
      }
      return bestMove;
    }
}

const evalBoard = (game) => {
    let totalEval = 0;
    const boardSize = 8;
    let i = 0, j = 0;
    for (let k = 0; k < boardSize * boardSize; k++) {
      //console.log("("+i+","+j+")");
      const square = game.SQUARES[k];
      const piece = game.get(square);
      if (piece !== null) {
        const val = getPieceValue(piece, i, j);
        totalEval = totalEval + val;
      }
      if (j === 7) {
        j = 0;
        i = i + 1;
      } else {
        j = j + 1;
      }
    }
    return totalEval;
}
// end AI

const onSnapEnd = () => {
    board.position(game.fen());
    if (game.turn() === "b") {
	     playMoveForBlack(game);
    }
}

const updateStatus = () => {
    let status = "";
    let moveColor = "White";
    if (game.turn() === 'b') {
	     moveColor = "Black";
    }
    if (game.in_checkmate()) {
	     status = "Game over, " + moveColor + " is in checkmate.";
    } else if (game.in_draw()) {
	     status = "Game over, drawn position";
    } else {
	     status = moveColor + " to move";
	     if (game.in_check()) {
	        status += ", " + moveColor + " is in check";
	     }
    }
    statusEl.html(status);
    fenEl.html(game.fen());
    pgnEl.html(game.pgn());
}

const config = {
    draggable: true,
    position: 'start',
    onDragStart: onDragStart,
    onDrop: onDrop,
    onSnapEnd: onSnapEnd
};

const init = () => {
    board = ChessBoard('board1', config);
    statusEl = $("#status");
    fenEl = $("#fen");
    pgnEl = $("#pgn");
};

$(document).ready(init);
