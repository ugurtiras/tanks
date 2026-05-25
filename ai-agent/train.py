import os
import threading
from stable_baselines3 import PPO
from env import TanksSingleAgentEnv, run_server, env

def train():
    model_path = "ppo_tanks_model"

    
    if os.path.exists(model_path + ".zip"):
        print("[TRAINING] Found existing model, resuming training...")
        model = PPO.load(model_path, env=env)
    else:
        print("[TRAINING] Creating a brand new PPO model...")
        model = PPO(
            "MlpPolicy",
            env,
            verbose=1,
            learning_rate=0.0003,
            n_steps=2048,
            batch_size=64,
            n_epochs=10,
            gamma=0.99
        )

    print("[TRAINING] PPO training loop starting...")
    
    try:
        model.learn(total_timesteps=500000, reset_num_timesteps=False)
        model.save(model_path)
        print(f"[TRAINING] Model saved successfully to {model_path}")
    except KeyboardInterrupt:
        print("\n[TRAINING] Training interrupted by user. Saving model...")
        model.save(model_path)
        print(f"[TRAINING] Model saved successfully to {model_path}")

if __name__ == "__main__":
    server_thread = threading.Thread(target=run_server, daemon=True)
    server_thread.start()
    print("[AI AGENT] Flask server listening on port 5000...")
    
    train()