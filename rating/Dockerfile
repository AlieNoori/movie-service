FROM debian:bookworm-slim

# Create and set working directory
WORKDIR /app

# Copy our binary and configs to the container
COPY main ./

# Make sure the binary is executable
RUN chmod +x ./main

# Expose the port for accepting incoming requests
EXPOSE 8082

# Execute our service
CMD ["/app/main"]
