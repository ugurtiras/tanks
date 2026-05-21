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
        
        self.observation_space = spaces.Box(
            low=0, high=800, shape=(16,), dtype=np.float32
        )
        
        self.current_obs = None
        self.next_action = None
        
        self.data_ready = threading.Event()
        self.action_ready = threading.Event()

    def reset(self, seed=None, options=None):
        super().reset(seed=seed)
        self.data_ready.wait()
        state = self._parse_observation(self.current_obs)
        self.data_ready.clear()
        return state, {}

    def step(self, action):
        self.next_action = action
        self.action_ready.set()
        
        self.data_ready.wait()
        state = self._parse_observation(self.current_obs)
        self.data_ready.clear()
        
        reward = 0.1 
        terminated = self.current_obs.get("isGameOver", False)
        
        return state, reward, terminated, False, {}

    def _parse_observation(self, raw_data):
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
    
    env.data_ready.set()
    env.action_ready.wait()
    env.action_ready.clear()
    
    action_mapping = {0: "UP", 1: "DOWN", 2: "LEFT", 3: "RIGHT", 4: "FIRE", 5: "STOP"}
    chosen_act = action_mapping[env.next_action]
    
    return jsonify({"action": chosen_act})

def run_server():
    app.run(port=5000, host='0.0.0.0', threaded=True)

if __name__ == "__main__":
    server_thread = threading.Thread(target=run_server, daemon=True)
    server_thread.start()
    print("[AI AGENT] Flask server listening on port 5000...")
    
    env.data_ready.wait()
    print("[AI AGENT] Bridge connected. Starting simulation...")
    
    while True:
        random_action = env.action_space.sample()
        state, reward, done, _, _ = env.step(random_action)
        if done:
            env.reset()