FROM node:20-alpine AS build

WORKDIR /app

COPY package.json package-lock.json ./

RUN npm ci --legacy-peer-deps && \
    ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        npm install --legacy-peer-deps @rollup/rollup-linux-x64-musl; \
    elif [ "$ARCH" = "aarch64" ]; then \
        npm install --legacy-peer-deps @rollup/rollup-linux-arm64-musl; \
    fi

COPY . .

RUN npm run build

FROM nginx:1.27-alpine

COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
