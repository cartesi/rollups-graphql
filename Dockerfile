# syntax=docker/dockerfile:1
# Use uma imagem base que tenha o Go instalado
FROM golang:1.23.1-bookworm

# Instale Clang
RUN apt-get update && apt-get install -y clang

ENV POSTGRES_HOST=db
ENV POSTGRES_PORT=5432
ENV POSTGRES_DB=mydatabase
ENV POSTGRES_USER=myuser
ENV POSTGRES_PASSWORD=mypassword

# Crie um diretório de trabalho fora do GOPATH
WORKDIR /app

# Copie os arquivos go.mod e go.sum para o diretório de trabalho
COPY ../go.mod ../go.sum ./

# Baixe as dependências
RUN go mod download

# Copie o resto dos arquivos do projeto para o diretório de trabalho
COPY ../ ./

# Execute o build da aplicação
RUN go build -o cartesi-rollups-graphql

# Exponha a porta em que a aplicação irá rodar
EXPOSE 8080

HEALTHCHECK CMD curl --fail http://localhost:8080/health || exit 1

# Comando para rodar a aplicação
CMD ["./cartesi-rollups-graphql", "--http-address=0.0.0.0", "--enable-debug", "--db-implementation", "postgres"]