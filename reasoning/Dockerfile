FROM python:3.12-slim

WORKDIR /app

COPY pyproject.toml .
RUN pip install --no-cache-dir .

COPY src/ src/

ENV REASONING_HOST=0.0.0.0
ENV REASONING_PORT=8000

EXPOSE 8000

CMD ["python", "-m", "uvicorn", "reasoning.app:app", "--host", "0.0.0.0", "--port", "8000"]
