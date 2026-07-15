FROM nginxinc/nginx-unprivileged:1.29-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY index.html styles.css /usr/share/nginx/html/
COPY robots.txt sitemap.xml site.webmanifest favicon.svg /usr/share/nginx/html/
COPY social-card.png icon-192.png icon-512.png apple-touch-icon.png /usr/share/nginx/html/

EXPOSE 8080
