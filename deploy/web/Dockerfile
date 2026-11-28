# syntax=docker/dockerfile:1
# SPDX-License-Identifier: GPL-2.0-only

FROM nginxinc/nginx-unprivileged:1.27-alpine

COPY deploy/web/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY web/index.html web/styles.css web/main.js web/orchestrator.js /usr/share/nginx/html/
COPY web/pkg /usr/share/nginx/html/pkg

EXPOSE 8080
