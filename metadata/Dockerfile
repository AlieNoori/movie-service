FROM debian:bookworm-slim

# Create and set working directory
WORKDIR /app

# Copy our binary and configs to the container
COPY main /app
COPY configs/default.yaml /app

# Make sure the binary is executable
RUN chmod +x /app/main

# Expose the port for accepting incoming requests
EXPOSE 8081

# Execute our service
CMD ["/app/main"]
