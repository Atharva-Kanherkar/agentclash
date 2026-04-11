import os


class Settings:
    host: str = os.getenv("REASONING_HOST", "0.0.0.0")
    port: int = int(os.getenv("REASONING_PORT", "8000"))
    log_level: str = os.getenv("REASONING_LOG_LEVEL", "info")


settings = Settings()
