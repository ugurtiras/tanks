from flask import Flask, request, jsonify
from stable_baselines3 import PPO
import numpy as np
import logging


log = logging.getLogger('werkzeug')
log.setLevel(logging.ERROR)

app = Flask(__name__)

print("Model loading...")
model = PPO.load("ppo_tanks_model")
print("Model loaded successfully!")

def parse_observation_production(raw_data):
   
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

@app.route('/act', methods=['POST'])
def act():
   
    raw_data = request.json
    
    state = parse_observation_production(raw_data)
    
    
    action, _states = model.predict(state, deterministic=False)
    
    action_mapping = {0: "UP", 1: "DOWN", 2: "LEFT", 3: "RIGHT", 4: "FIRE", 5: "STOP"}
    chosen_act = action_mapping[int(action)]
    
   
    return jsonify({"action": chosen_act})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, threaded=True)