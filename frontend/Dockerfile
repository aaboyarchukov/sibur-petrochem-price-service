# syntax=docker/dockerfile:1

FROM node:20-alpine AS builder
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

# Кнопка «Загрузить демо-набор» показывается только при DEMO_ENABLED=true (флаг сборки)
ARG VITE_DEMO_ENABLED=false
ENV VITE_DEMO_ENABLED=$VITE_DEMO_ENABLED

COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8080
ENTRYPOINT ["nginx", "-g", "daemon off;"]
