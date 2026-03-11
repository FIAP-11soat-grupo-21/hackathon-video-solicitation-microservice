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

// Cria a tabela Videos no DynamoDB
func CreateVideosTable(ctx context.Context, db *dynamodb.Client) error {
	fmt.Println("[DEBUG] CreateVideosTable: preparando input...")
	input := &dynamodb.CreateTableInput{
		TableName: aws.String("videos-05"),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("user_id"),
				AttributeType: "S",
			},
			{
				AttributeName: aws.String("id"),
				AttributeType: "S",
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("user_id"),
				KeyType:       "HASH",
			},
			{
				AttributeName: aws.String("id"),
				KeyType:       "RANGE",
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
	}
	fmt.Printf("[DEBUG] CreateVideosTable: input = %+v\n", input)
	fmt.Printf("[DEBUG] CreateVideosTable: chamando db.CreateTable para tabela %s\n", *input.TableName)
	_, err := db.CreateTable(ctx, input)
	if err != nil {
		fmt.Printf("[DEBUG] CreateVideosTable: erro ao criar tabela %s: %v\n", *input.TableName, err)
		if err.Error() != "" {
			fmt.Printf("[DEBUG] CreateVideosTable: erro detalhado: %v\n", err.Error())
		}
		return fmt.Errorf("failed to create Videos table %s: %w", *input.TableName, err)
	}
	fmt.Printf("[DEBUG] CreateVideosTable: tabela %s criada com sucesso!\n", *input.TableName)
	return nil
}

func RunMigrationsDynamoDB(db *dynamodb.Client, ctx context.Context) error {
	log.Println("[MIGRATION] Iniciando criação da tabela Videos...")
	if err := CreateVideosTable(ctx, db); err != nil {
		log.Printf("[MIGRATION] Erro ao criar tabela Videos: %v", err)
		return err
	}
	log.Println("[MIGRATION] Tabela Videos criada com sucesso!")
	return nil
}

// Adicione funções auxiliares para criar índices LSI/GSI conforme necessário.
