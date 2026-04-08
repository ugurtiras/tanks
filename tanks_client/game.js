import { Renderer } from './renderer.js';

const canvas = document.getElementById('gameCanvas');
const ctx = canvas.getContext('2d');
const renderer = new Renderer(ctx);

const lobby = document.getElementById('lobby-screen');
const gameScreen = document.getElementById('game-screen');
const nickInput = document.getElementById('nick-input');
const roomInput = document.getElementById('room-input');
const joinBtn = document.getElementById('join-btn');
const createBtn = document.getElementById('create-btn');
const lobbyStatus = document.getElementById('lobby-status');
const roomIdLabel = document.getElementById('room-id');
const ownerNickLabel = document.getElementById('owner-nick');
const roomLobbyOverlay = document.getElementById('room-lobby-overlay');
const roomLobbyRoomId = document.getElementById('room-lobby-room-id');
const roomOwner = document.getElementById('room-owner');
const roomPlayers = document.getElementById('room-players');
const roomPlayerCount = document.getElementById('room-player-count');
const roomLobbyStatus = document.getElementById('room-lobby-status');
const startGameBtn = document.getElementById('start-game-btn');
const resultOverlay = document.getElementById('result-overlay');
const resultText = document.getElementById('result-text');
const backToLobbyBtn = document.getElementById('back-to-lobby-btn');

let socket = null;
let nickname = "";
let roomId = "";
let gameState = { type: "GAME_STATE", players: {}, bullets: [] };
let isJoined = false;
let isOwner = false;
let gameStarted = false;
let gameOver = false;
let roomState = {
    owner: "",
    players: [],
    started: false,
    gameOver: false,
    winner: "",
    maxPlayers: 4
};
let lastInputPayload = null;
let pendingFire = false;
const keys = {};
const roomCodeRegex = /^[A-Z]{7}$/;

window.onkeydown = (e) => {
    const wasPressed = !!keys[e.code];
    keys[e.code] = true;
    if (e.code === 'Space') {
        e.preventDefault();
        if (!wasPressed) {
            pendingFire = true;
        }
    }
};
window.onkeyup = (e) => { keys[e.code] = false; };

joinBtn.disabled = true;
createBtn.disabled = true;
lobbyStatus.innerText = 'Sunucuya baglaniyor...';

roomInput.addEventListener('input', () => {
    roomInput.value = roomInput.value.toUpperCase().replace(/[^A-Z]/g, '').slice(0, 7);
});

function getWsUrl() {
    const host = window.location.hostname || 'localhost';
    return `ws://${host}:8080/ws`;
}

function updatePlayerCount(count) {
    document.getElementById('count').innerText = String(count);
}

function renderRoomPlayers(players, owner) {
    roomPlayers.innerHTML = "";
    players.forEach((p) => {
        const li = document.createElement('li');
        li.innerText = p === owner ? `${p} (admin)` : p;
        roomPlayers.appendChild(li);
    });
}

function updateLobbyOverlay() {
    roomLobbyRoomId.innerText = roomId || '-';
    roomOwner.innerText = roomState.owner || '-';
    ownerNickLabel.innerText = roomState.owner || '-';
    roomPlayerCount.innerText = String((roomState.players || []).length);
    renderRoomPlayers(roomState.players || [], roomState.owner || "");

    const canStart = isOwner && !roomState.started && (roomState.players || []).length >= 2;
    startGameBtn.disabled = !canStart;
    startGameBtn.style.display = isOwner ? 'block' : 'none';

    if (roomState.started) {
        roomLobbyOverlay.style.display = 'none';
        roomLobbyStatus.innerText = '';
        return;
    }

    roomLobbyOverlay.style.display = 'flex';
    resultOverlay.style.display = roomState.gameOver ? 'flex' : 'none';

    if (!isOwner) {
        roomLobbyStatus.innerText = 'Admin oyunu baslatinca tur baslayacak.';
    } else if ((roomState.players || []).length < 2) {
        roomLobbyStatus.innerText = 'Oyunu baslatmak icin en az 2 oyuncu gerekli.';
    } else {
        roomLobbyStatus.innerText = 'Hazirsan oyunu baslatabilirsin.';
    }
}

function showResult(winner) {
    gameOver = true;
    gameStarted = false;
    roomState.gameOver = true;
    roomState.started = false;
    roomState.winner = winner || "";
    resultText.innerText = winner ? `Kazanan: ${winner}` : 'Kazanan yok';
    resultOverlay.style.display = 'flex';
    updateLobbyOverlay();
}

function connect() {
    socket = new WebSocket(getWsUrl());

    socket.onopen = () => {
        joinBtn.disabled = false;
        createBtn.disabled = false;
        lobbyStatus.innerText = 'Baglandi. Nick girip katilabilirsin.';
        lobbyStatus.className = 'lobby-status connected';
    };

    socket.onerror = () => {
        lobbyStatus.innerText = 'Baglanti hatasi.';
        lobbyStatus.className = 'lobby-status error';
    };

    socket.onclose = () => {
        joinBtn.disabled = true;
        createBtn.disabled = true;

        if (!isJoined) {
            lobbyStatus.innerText = 'Baglanti kapandi.';
            lobbyStatus.className = 'lobby-status error';
        }
    };

    socket.onmessage = onSocketMessage;
}

function validateNickname(name) {
    return /^[A-Za-z0-9_]{3,12}$/.test(name);
}

function sendLogin(action) {
    nickname = nickInput.value.trim();
    const enteredRoom = roomInput.value.trim().toUpperCase();

    if (!validateNickname(nickname)) {
        lobbyStatus.innerText = 'Nick 3-12 karakter olmali (harf/rakam/_).';
        lobbyStatus.className = 'lobby-status error';
        return;
    }

    if (action === 'join' && !roomCodeRegex.test(enteredRoom)) {
        lobbyStatus.innerText = 'Oda kodu 7 buyuk harf olmali.';
        lobbyStatus.className = 'lobby-status error';
        return;
    }

    if (socket.readyState !== WebSocket.OPEN) {
        lobbyStatus.innerText = 'Henuz bagli degil. Biraz bekle.';
        lobbyStatus.className = 'lobby-status error';
        return;
    }

    socket.send(JSON.stringify({
        type: "LOGIN",
        nickname,
        action,
        roomId: action === 'join' ? enteredRoom : ""
    }));
}

joinBtn.onclick = () => sendLogin('join');
createBtn.onclick = () => sendLogin('create');
startGameBtn.onclick = () => sendJson({ type: 'START_GAME' });
backToLobbyBtn.onclick = () => returnToLobby();

function returnToLobby() {
    gameStarted = false;
    gameOver = false;
    pendingFire = false;
    lastInputPayload = null;
    Object.keys(keys).forEach((k) => delete keys[k]);

    resultText.innerText = 'Kazanan bekleniyor...';
    roomState.gameOver = false;
    roomState.started = false;
    roomState.winner = "";
    resultOverlay.style.display = 'none';
    roomLobbyOverlay.style.display = 'flex';
    updateLobbyOverlay();
}

function onSocketMessage(event) {
    const msg = JSON.parse(event.data);

    if (msg.type === "AUTH_SUCCESS") {
        isJoined = true;
        roomId = (msg.roomId || '').toUpperCase();
        isOwner = !!msg.isOwner;
        lobby.style.display = "none";
        gameScreen.style.display = "block";
        document.getElementById('player-nick').innerText = nickname;
        roomIdLabel.innerText = roomId || '-';
        roomLobbyRoomId.innerText = roomId || '-';
        updateLobbyOverlay();
        updatePlayerCount(Object.keys(gameState.players || {}).length);
        requestAnimationFrame(gameLoop);
    }

    if (msg.type === "AUTH_FAIL") {
        lobbyStatus.innerText = msg.message || 'Giris basarisiz.';
        lobbyStatus.className = 'lobby-status error';
    }

    if (msg.type === "GAME_STATE") {
        gameState = msg;
        gameStarted = !!msg.started;
        gameOver = !!msg.gameOver;
        updatePlayerCount(Object.keys(msg.players || {}).length);
    }

    if (msg.type === "PLAYER_LIST" && Array.isArray(msg.players)) {
        updatePlayerCount(msg.players.length);
    }

    if (msg.type === "ROOM_STATE") {
        roomState = {
            owner: msg.owner || "",
            players: Array.isArray(msg.players) ? msg.players : [],
            started: !!msg.started,
            gameOver: !!msg.gameOver,
            winner: msg.winner || "",
            maxPlayers: msg.maxPlayers || 4
        };
        if (roomState.gameOver) {
            showResult(roomState.winner);
        } else {
            if (roomState.started) {
                gameStarted = true;
                gameOver = false;
                resultOverlay.style.display = 'none';
            }
            updateLobbyOverlay();
        }
    }

    if (msg.type === "GAME_STARTED") {
        gameStarted = true;
        gameOver = false;
        roomState.started = true;
        roomState.gameOver = false;
        resultOverlay.style.display = 'none';
        updateLobbyOverlay();
    }

    if (msg.type === "GAME_OVER") {
        showResult(msg.winner || "");
    }
}

function sendJson(payload) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        return;
    }
    socket.send(JSON.stringify(payload));
}

function sendInput() {
    if (!isJoined || !gameStarted || gameOver) return;

    const inputPayload = {
        type: "INPUT",
        nickname: nickname,
        up: !!(keys['KeyW'] || keys['ArrowUp']),
        down: !!(keys['KeyS'] || keys['ArrowDown']),
        left: !!(keys['KeyA'] || keys['ArrowLeft']),
        right: !!(keys['KeyD'] || keys['ArrowRight']),
        fire: pendingFire
    };

    const serializedInput = JSON.stringify(inputPayload);
    if (serializedInput !== lastInputPayload || pendingFire) {
        sendJson(inputPayload);
        lastInputPayload = serializedInput;
    }
    pendingFire = false;
}

function gameLoop() {
    if (!isJoined) return;
    sendInput();
    renderer.draw(gameState, nickname);
    requestAnimationFrame(gameLoop);
}

connect();