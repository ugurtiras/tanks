import gymnasium as gym
from gymnasium import spaces
import numpy as np
from flask import Flask, request, jsonify
import threading
import logging

log = logging.getLogger('werkzeug')
log.setLevel(logging.ERROR)

class TanksSingleAgentEnv(gym.Env):
    def __init__(self):
        super(TanksSingleAgentEnv, self).__init__()
        
        self.action_space = spaces.Discrete(6)
        
        # Go haritan büyükse high değerini harita boyutuna (örn: 800) çekebilirsin
        self.observation_space = spaces.Box(
            low=-1000, high=1000, shape=(16,), dtype=np.float32
        )
        
        self.current_obs = None
        self.next_action = None
        
        self.data_ready = threading.Event()
        self.action_ready = threading.Event()
        
        # Kilitlenmeyi önlemek için ilk reset kontrolü
        self.is_first_reset = True

    def reset(self, seed=None, options=None):
        super().reset(seed=seed)
        
        # Eğer bu PPO'nun ilk reset isteğiyse ve Go henüz veri atmadıysa bekle
        if self.is_first_reset and self.current_obs is None:
            print("[ENV] Waiting for Go server to send initial game state...")
            self.data_ready.wait()
            self.is_first_reset = False
            
        state = self._parse_observation(self.current_obs)
        return state, {}

    def step(self, action):
        self.next_action = action
        self.action_ready.set()  # Flask'a "aksiyon hazır, Go'ya dön" diyoruz
        
        # Go'nun bir sonraki kareyi hesaplayıp yeni veriyi basmasını bekliyoruz
        self.data_ready.wait()
        self.data_ready.clear()
        
        state = self._parse_observation(self.current_obs)
        
        # --- TEK ATISTA ÖLÜM MODU - ÖDÜL VE CEZA AYARI ---
        reward = 0.0
        
        # 1. Hayatta kalınan her an için minik motivasyon (Mermilerden kaçmayı tetikler)
        reward += 0.05 
        
        # 2. Düşmanı vurduysa (Yani tek atışta onu yok ettiyse)
        if self.current_obs.get("hitEnemy", False) or self.current_obs.get("killedEnemy", False):
            reward += 30.0  
            print("[REWARD] Bot shot and killed an enemy! +30")
            
        # 3. Kendisi mermi yiyip öldüyse (Raunt bittiyse)
        terminated = self.current_obs.get("isGameOver", False)
        if terminated:
            reward -= 25.0  
            print("[PENALTY] Bot got one-shotted! Game Over. -25")
        
        return state, reward, terminated, False, {}

    def _parse_observation(self, raw_data):
        if raw_data is None:
            return np.zeros(16, dtype=np.float32)
            
        players = raw_data.get("players", [])
        current_name = raw_data.get("current_turn_player", "")
        
        me = next((p for p in players if p["name"] == current_name), None)
        if not me:
            return np.zeros(16, dtype=np.float32)
            
        enemies = [p for p in players if p["name"] != current_name]
        enemies.sort(key=lambda e: (e["x"] - me["x"])**2 + (e["y"] - me["y"])**2)
        
        obs = [me["x"], me["y"], me["angle"], me["health"]]
        
        for i in range(3):
            if i < len(enemies):
                e = enemies[i]
                obs.extend([e["x"], e["y"], e["angle"], e["health"]])
            else:
                obs.extend([0.0, 0.0, 0.0, 0.0])
                
        return np.array(obs, dtype=np.float32)

app = Flask(__name__)
env = TanksSingleAgentEnv()

@app.route('/act', methods=['POST'])
def act():
    env.current_obs = request.json
    
    # Yeni veri geldi, step fonksiyonunu uyandır
    env.data_ready.set()
    
    # PPO'nun step fonksiyonunun yeni bir karar (action) üretmesini bekle
    env.action_ready.wait()
    env.action_ready.clear()
    
    action_mapping = {0: "UP", 1: "DOWN", 2: "LEFT", 3: "RIGHT", 4: "FIRE", 5: "STOP"}
    chosen_act = action_mapping[env.next_action]
    
    return jsonify({"action": chosen_act})

def run_server():
    app.run(port=5000, host='0.0.0.0', threaded=True)

# train.py bu dosyayı import edeceği için alttaki eski random test döngüsünü sildik!