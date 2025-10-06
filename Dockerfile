# ---- Étape 1 : build de l'application Go ----
FROM golang:1.25-alpine AS build

# répertoire de travail dans le container
WORKDIR /app

# copier les fichiers de dépendances
COPY go.mod go.sum ./
RUN go mod download

# Installe GTK (nécessaire pour github.com/sqweek/dialog)
RUN apk add --no-cache gtk+3.0-dev glib-dev pkgconfig

# copier le reste du projet
COPY . .

# compiler ton app (binaire statique)
RUN go build -o /mys3

# ---- Étape 2 : image finale minimale ----
FROM alpine:3.18

# ajouter les certificats SSL (utile pour HTTPS si jamais)
RUN apk add --no-cache ca-certificates

# copier le binaire depuis l'étape build
COPY --from=build /mys3 /mys3

# exposer le port de ton API Gin
EXPOSE 8080

# commande de démarrage
CMD ["/mys3"]
