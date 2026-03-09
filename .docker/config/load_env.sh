#!/bin/sh

# -----------------------------------------------------------
# Configurações
# -----------------------------------------------------------
# O caminho onde o Secrets Store CSI Driver montou os secrets.
# DEVE ser o mesmo valor de 'volumeMounts.mountPath' no seu Deployment.
SECRET_MOUNT_PATH="/mnt/secrets"

# O nome do arquivo .env que será gerado.
ENV_FILE_NAME=".env"

# -----------------------------------------------------------
# 1. Verifica se o diretório de secrets existe
# -----------------------------------------------------------
if [ ! -d "$SECRET_MOUNT_PATH" ]; then
    echo "ERRO: O diretório de secrets ($SECRET_MOUNT_PATH) não foi encontrado."
    echo "Certifique-se de que o volume CSI do Secrets Store foi montado corretamente."
    exit 1
fi

# -----------------------------------------------------------
# 2. Inicializa e limpa o arquivo .env
# -----------------------------------------------------------
echo "# Arquivo .env gerado automaticamente a partir do Google Secret Manager" > "$ENV_FILE_NAME"
echo "" >> "$ENV_FILE_NAME"

# -----------------------------------------------------------
# 3. Varre o diretório e cria o arquivo .env
# -----------------------------------------------------------
echo "Processando secrets em $SECRET_MOUNT_PATH..."

# Itera sobre cada arquivo (secret) dentro do diretório montado
# O nome do arquivo será usado como o nome da variável (em maiúsculas)
for secret_file in "$SECRET_MOUNT_PATH"/*; do
    # Garante que estamos processando apenas arquivos
    if [ -f "$secret_file" ]; then
        # Extrai o nome do arquivo (a "chave")
        # Ex: /mnt/secrets/DB_PASSWORD vira DB_PASSWORD
        key=$(basename "$secret_file")

        # Converte a chave para MAIÚSCULAS (prática comum para variáveis de ambiente)
        # Nota: O nome da variável de ambiente não pode ter hífens ou caracteres especiais.
        # Se seus 'fileName' no SecretProviderClass usarem hífens, substitua-os aqui por underscores.
        env_key=$(echo "$key" | tr '[:lower:]-' '[:upper:]_')

        # Lê o valor do arquivo (o "valor" do secret), remove quebras de linha (strip)
        # e escapa as aspas simples para segurança ao injetar.
        value=$(cat "$secret_file" | tr -d '\n' | sed "s/'/'\\\''/g")

        # Adiciona a linha KEY='VALUE' ao arquivo .env
        echo "$env_key='$value'" >> "$ENV_FILE_NAME"
        echo " -> Adicionado: $env_key"
    fi
done

echo ""
echo "Sucesso! O arquivo $ENV_FILE_NAME foi criado."
echo "Conteúdo de $ENV_FILE_NAME:"
cat "$ENV_FILE_NAME"

# Este arquivo .env agora pode ser lido pela sua aplicação ou injetado
# no contêiner principal usando ferramentas como 'source .env' ou bibliotecas dotenv.