FROM oven/bun:1.3.5-alpine

WORKDIR /app
ENV NODE_ENV=production

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile --production

COPY . .

EXPOSE 3000
CMD ["bun", "start"]
