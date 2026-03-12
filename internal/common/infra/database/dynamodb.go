// Certifique-se de que o client DynamoDB está usando o endpoint correto (http://dynamodb:8000)
// e que RunMigrationsDynamoDB está sendo chamado na inicialização da aplicação.

package database

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func NewDynamoDB(cfg aws.Config) (*dynamodb.Client, error) {
	client := dynamodb.NewFromConfig(cfg)
	return client, nil
}

func CreateVideosTable(ctx context.Context, db *dynamodb.Client) error {

	// Esta tabela armazena as solicitações de processamento de vídeos
	// do microserviço responsável pelo domínio de vídeos.

	input := &dynamodb.CreateTableInput{
		TableName: aws.String("videos"),

		// AttributeDefinitions define apenas atributos usados em chaves
		// (Primary Key ou índices). Diferente de banco relacional,
		// não precisamos declarar todas as colunas da tabela.
		AttributeDefinitions: []types.AttributeDefinition{

			// user_id será a Partition Key da tabela.
			// Isso permite listar todos os vídeos de um usuário com eficiência.
			{
				AttributeName: aws.String("user_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},

			// video_id será a Sort Key da tabela.
			// Cada vídeo de um usuário terá um identificador único.
			{
				AttributeName: aws.String("video_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},

			// status será usado em um índice secundário
			// para permitir buscar vídeos por status (ex: PENDING, PROCESSING).
			{
				AttributeName: aws.String("status"),
				AttributeType: types.ScalarAttributeTypeS,
			},

			// created_at será usado como Sort Key no índice de status
			// para ordenar os vídeos por data de criação.
			{
				AttributeName: aws.String("created_at"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},

		// Definição da chave primária da tabela
		KeySchema: []types.KeySchemaElement{

			// Partition Key
			// Permite executar queries eficientes como:
			// GET /videos/user/:user_id
			{
				AttributeName: aws.String("user_id"),
				KeyType:       types.KeyTypeHash,
			},

			// Sort Key
			// Diferencia os vídeos de um mesmo usuário
			{
				AttributeName: aws.String("video_id"),
				KeyType:       types.KeyTypeRange,
			},
		},

		// Modo serverless (sem provisionamento de capacidade)
		BillingMode: types.BillingModePayPerRequest,

		// Índices secundários globais
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{

			// ---------------------------------------------------------
			// GSI 1: Buscar vídeo diretamente pelo ID
			// ---------------------------------------------------------
			// Permite buscar um vídeo específico sem conhecer o user_id
			// Exemplo de uso:
			// GET /videos/:video_id
			{
				IndexName: aws.String("video_id-index"),

				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("video_id"),
						KeyType:       types.KeyTypeHash,
					},
				},

				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
			},

			// ---------------------------------------------------------
			// GSI 2: Buscar vídeos por status
			// ---------------------------------------------------------
			// Muito útil para workers que processam vídeos.
			// Exemplo:
			// buscar vídeos com status PENDING ordenados por data.
			{
				IndexName: aws.String("status-index"),

				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("status"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("created_at"),
						KeyType:       types.KeyTypeRange,
					},
				},

				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
			},
		},
	}

	// Cria a tabela
	_, err := db.CreateTable(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create Videos table %s: %w", *input.TableName, err)
	}

	fmt.Printf("Tabela %s criada com sucesso\n", *input.TableName)
	return nil
}

func CreateChunksTable(ctx context.Context, db *dynamodb.Client) error {
	fmt.Println("[DEBUG] CreateChunksTable: preparando input...")
	input := &dynamodb.CreateTableInput{
		TableName: aws.String("chunks"),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("video_id"),
				AttributeType: "S",
			},
			{
				AttributeName: aws.String("part_number"),
				AttributeType: "N",
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("video_id"),
				KeyType:       "HASH",
			},
			{
				AttributeName: aws.String("part_number"),
				KeyType:       "RANGE",
			},
		},
		BillingMode: types.BillingModePayPerRequest,
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("video_id-index"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("video_id"),
						KeyType:       "HASH",
					},
					{
						AttributeName: aws.String("part_number"),
						KeyType:       "RANGE",
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
			},
		},
	}
	fmt.Printf("[DEBUG] CreateChunksTable: input = %+v\n", input)
	fmt.Printf("[DEBUG] CreateChunksTable: chamando db.CreateTable para tabela %s\n", *input.TableName)
	_, err := db.CreateTable(ctx, input)
	if err != nil {
		fmt.Printf("[DEBUG] CreateChunksTable: erro ao criar tabela %s: %v\n", *input.TableName, err)
		return fmt.Errorf("failed to create Chunks table %s: %w", *input.TableName, err)
	}
	fmt.Printf("[DEBUG] CreateChunksTable: tabela %s criada com sucesso!\n", *input.TableName)
	return nil
}

func RunMigrationsDynamoDB(db *dynamodb.Client, ctx context.Context) error {
	log.Println("[MIGRATION] Iniciando criação da tabela Videos...")
	if err := CreateVideosTable(ctx, db); err != nil {
		log.Printf("[MIGRATION] Erro ao criar tabela Videos: %v", err)
		return err
	}
	log.Println("[MIGRATION] Tabela Videos criada com sucesso!")
	log.Println("[MIGRATION] Iniciando criação da tabela Chunks...")
	if err := CreateChunksTable(ctx, db); err != nil {
		log.Printf("[MIGRATION] Erro ao criar tabela Chunks: %v", err)
		return err
	}
	log.Println("[MIGRATION] Tabela Chunks criada com sucesso!")
	return nil
}
