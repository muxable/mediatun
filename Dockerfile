FROM node:14.18.1 AS nodebuilder

ENV NODE_ENV=production

WORKDIR /app

COPY ["package.json", "package-lock.json", "./"]

RUN npm ci

COPY . .

RUN npm run build

FROM caddy:2-builder-alpine AS caddybuilder

RUN xcaddy build --with github.com/mholt/caddy-l4

FROM caddy:2-alpine
COPY --from=caddybuilder /usr/bin/caddy /usr/bin/caddy
COPY Caddyfile.json /etc/caddy/Caddyfile.json
COPY --from=nodebuilder /app/build /var/www/html

CMD ["caddy", "run", "-config", "/etc/caddy/Caddyfile.json"]