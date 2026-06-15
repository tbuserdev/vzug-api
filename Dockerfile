FROM oven/bun:1.3.5-alpine

WORKDIR /app

# Install dependencies
COPY package.json bun.lock ./
RUN bun install

# Copy everything (including public/ and index.ts)
COPY . .

# Expose and Run directly from TS (Bun's strength)
EXPOSE 3000
CMD ["bun", "run", "index.ts"]