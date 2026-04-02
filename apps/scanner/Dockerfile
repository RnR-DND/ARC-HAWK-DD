# Use the official Python image as the base image
FROM python:3.11-slim

# Set the working directory in the container
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    libsm6 \
    libxext6 \
    tesseract-ocr \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy the requirements first for caching
COPY requirements.txt /app/
RUN pip3 install --no-cache-dir -r requirements.txt

# Install Flask for API
RUN pip3 install --no-cache-dir flask requests

ENV PYTHONUNBUFFERED=1

# Copy the local code to the container
COPY . /app/

# Install the scanner package
RUN pip3 install -e .

# Expose the API port
EXPOSE 5002

# Environment variables
ENV PORT=5002
ENV BACKEND_URL=http://backend:8080

# H-6: Run as non-root user
RUN useradd -m scanner
USER scanner

# Run the HTTP API server
CMD ["python3", "scanner_api.py"]