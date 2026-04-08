import { TILE_SIZE, MAZE, MAP_SIZE } from './constants.js';

export class Renderer {
    constructor(ctx) {
        this.ctx = ctx;
    }

    isBoundaryCell(x, y) {
        return x === 0 || y === 0 || x === MAZE[0].length - 1 || y === MAZE.length - 1;
    }

    draw(gameState, myNickname) {
        // Ekranı temizle
        this.ctx.fillStyle = "#121212";
        this.ctx.fillRect(0, 0, MAP_SIZE, MAP_SIZE);

        this.drawMaze();

        // TANKLARI ÇİZ (Sadece Go'dan gelen koordinatlarla!)
        const players = (gameState && gameState.players) || {};
        Object.keys(players).forEach(id => {
                const p = players[id];
                if (p.health > 0) {
                    this.drawTank(p, id, myNickname);
                }
            });

        // MERMİLERİ ÇİZ
        const bullets = (gameState && Array.isArray(gameState.bullets)) ? gameState.bullets : [];
        bullets.forEach(b => this.drawBullet(b));
    }

    drawMaze() {
        this.ctx.strokeStyle = "#00FF41";
        this.ctx.lineWidth = 3;
        this.ctx.strokeRect(0, 0, MAP_SIZE, MAP_SIZE);
        this.drawHorizontalWallSegments();
        this.drawVerticalWallSegments();
    }

    drawHorizontalWallSegments() {
        for (let y = 0; y < MAZE.length; y++) {
            let startTop = -1;
            let startBottom = -1;

            for (let x = 0; x <= MAZE[y].length; x++) {
                const inBounds = x < MAZE[y].length;
                const isWall = inBounds && MAZE[y][x] === 1 && !this.isBoundaryCell(x, y);
                const topEdge = isWall && (y === 0 || MAZE[y - 1][x] === 0);
                const bottomEdge = isWall && (y === MAZE.length - 1 || MAZE[y + 1][x] === 0);

                if (topEdge && startTop === -1) {
                    startTop = x;
                }
                if ((!topEdge || !inBounds) && startTop !== -1) {
                    const yPx = y * TILE_SIZE;
                    this.line(startTop * TILE_SIZE, yPx, x * TILE_SIZE, yPx);
                    startTop = -1;
                }

                if (bottomEdge && startBottom === -1) {
                    startBottom = x;
                }
                if ((!bottomEdge || !inBounds) && startBottom !== -1) {
                    const yPx = (y + 1) * TILE_SIZE;
                    this.line(startBottom * TILE_SIZE, yPx, x * TILE_SIZE, yPx);
                    startBottom = -1;
                }
            }
        }
    }

    drawVerticalWallSegments() {
        const width = MAZE[0].length;
        const height = MAZE.length;

        for (let x = 0; x < width; x++) {
            let startLeft = -1;
            let startRight = -1;

            for (let y = 0; y <= height; y++) {
                const inBounds = y < height;
                const isWall = inBounds && MAZE[y][x] === 1 && !this.isBoundaryCell(x, y);
                const leftEdge = isWall && (x === 0 || MAZE[y][x - 1] === 0);
                const rightEdge = isWall && (x === width - 1 || MAZE[y][x + 1] === 0);

                if (leftEdge && startLeft === -1) {
                    startLeft = y;
                }
                if ((!leftEdge || !inBounds) && startLeft !== -1) {
                    const xPx = x * TILE_SIZE;
                    this.line(xPx, startLeft * TILE_SIZE, xPx, y * TILE_SIZE);
                    startLeft = -1;
                }

                if (rightEdge && startRight === -1) {
                    startRight = y;
                }
                if ((!rightEdge || !inBounds) && startRight !== -1) {
                    const xPx = (x + 1) * TILE_SIZE;
                    this.line(xPx, startRight * TILE_SIZE, xPx, y * TILE_SIZE);
                    startRight = -1;
                }
            }
        }
    }

    line(x1, y1, x2, y2) {
        this.ctx.beginPath();
        this.ctx.moveTo(x1, y1);
        this.ctx.lineTo(x2, y2);
        this.ctx.stroke();
    }

    drawTank(p, id, myNickname) {
        this.ctx.save();
        this.ctx.translate(p.x, p.y); // Go'dan gelen X ve Y!
        this.ctx.rotate(p.angle);
        this.ctx.fillStyle = (id === myNickname) ? "lime" : "red";
        this.ctx.fillRect(-12, -12, 24, 24);
        this.ctx.fillStyle = "white";
        this.ctx.fillRect(8, -2, 12, 4); // Namlu
        this.ctx.restore();
        
        // Nickname yazısı
        this.ctx.fillStyle = "white";
        this.ctx.font = "10px Arial";
        this.ctx.fillText(id, p.x - 10, p.y - 20);
    }

    drawBullet(b) {
        this.ctx.fillStyle = "yellow";
        this.ctx.beginPath();
        this.ctx.arc(b.x, b.y, 3, 0, Math.PI * 2);
        this.ctx.fill();
    }
}