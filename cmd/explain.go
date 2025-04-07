/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

var config Config

type Config struct {
	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string
	OpenAIAPIKey  string
	OpenAIBaseURL string
	OpenAIModel   string
}

type TableInfo struct {
	TableName   string
	Columns     []ColumnInfo
	Indexes     []IndexInfo
	CreateTable string
}

type ColumnInfo struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default sql.NullString
	Extra   string
}

type IndexInfo struct {
	IndexName  string
	ColumnName string
	NonUnique  int
}

type ExplainResult struct {
	ID           int
	SelectType   string
	Table        string
	Partitions   string
	Type         string
	PossibleKeys string
	Key          string
	KeyLen       string
	Ref          string
	Rows         int
	Filtered     float64
	Extra        string
}

type AnalysisRequest struct {
	SQLQuery    string
	TableInfos  []TableInfo
	ExplainPlan []ExplainResult
}

// explainCmd represents the explain command
var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "explain sql",
	Long:  `explain sql and give some advice`,
	Run: func(cmd *cobra.Command, args []string) {
		// 检查并加载.env文件
		if _, err := os.Stat(".env"); err != nil {
			fmt.Println("当前目录未存在.env文件")
			return
		}

		// 加载.env文件内容到config
		if err := godotenv.Load(); err != nil {
			fmt.Printf("加载.env文件失败: %v\n", err)
			return
		}

		// 设置config值
		config = Config{
			MySQLHost:     os.Getenv("host"),
			MySQLPort:     os.Getenv("port"),
			MySQLUser:     os.Getenv("username"),
			MySQLPassword: os.Getenv("password"),
			MySQLDatabase: os.Getenv("database"),
			OpenAIAPIKey:  os.Getenv("ai_api_key"),
			OpenAIBaseURL: os.Getenv("ai_base_url"),
			OpenAIModel:   os.Getenv("ai_model"),
		}

		if len(args) == 0 {
			cmd.Help()
			return
		}

		sql := args[0]
		if len(sql) >= 7 && (strings.HasPrefix(strings.ToUpper(sql), "EXPLAIN")) {
			sql = sql[7:]
		}
		sql = strings.TrimSpace(sql)

		// 连接MySQL
		db, err := connectMySQL(config)
		if err != nil {
			fmt.Printf("连接MySQL失败: %v\n", err)
			return
		}
		defer db.Close()

		// 获取SQL涉及的表
		tables := extractTablesFromSQL(sql)
		if len(tables) == 0 {
			fmt.Println("未检测到表名")
			return
		}

		// 收集表结构信息
		var tableInfos []TableInfo
		for _, table := range tables {
			info, err := getTableInfo(db, table)
			if err != nil {
				fmt.Printf("获取表%s信息失败: %v\n", table, err)
				continue
			}
			tableInfos = append(tableInfos, info)
		}

		// 执行EXPLAIN
		explainResults, err := executeExplain(db, sql)
		if err != nil {
			fmt.Printf("执行EXPLAIN失败: %v\n", err)
			return
		}

		// 准备AI请求
		request := AnalysisRequest{
			SQLQuery:    sql,
			TableInfos:  tableInfos,
			ExplainPlan: explainResults,
		}

		// 发送给AI分析
		analysis, err := sendToAI(config.OpenAIAPIKey, request)
		if err != nil {
			fmt.Printf("AI分析失败: %v\n", err)
			return
		}

		fmt.Println("\nAI分析结果:")
		fmt.Println(analysis)
	},
}

func init() {
	rootCmd.AddCommand(explainCmd)
}

func connectMySQL(config Config) (*sql.DB, error) {
	fmt.Println(config)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		config.MySQLUser,
		config.MySQLPassword,
		config.MySQLHost,
		config.MySQLPort,
		config.MySQLDatabase)
	fmt.Println(dsn)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func getTableInfo(db *sql.DB, tableName string) (TableInfo, error) {
	var info TableInfo
	info.TableName = tableName

	// 获取列信息
	rows, err := db.Query(fmt.Sprintf("DESCRIBE `%s`", tableName))
	if err != nil {
		return info, err
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var def sql.NullString
		err := rows.Scan(&col.Field, &col.Type, &col.Null, &col.Key, &def, &col.Extra)
		if err != nil {
			return info, err
		}
		col.Default = def
		info.Columns = append(info.Columns, col)
	}

	// 获取索引信息
	indexRows, err := db.Query(fmt.Sprintf("SHOW INDEX FROM `%s`", tableName))
	if err != nil {
		return info, err
	}
	defer indexRows.Close()

	for indexRows.Next() {
		var idx IndexInfo
		var dummy1, dummy2, dummy3, dummy4, dummy5, dummy6, dummy7, dummy8, dummy9, dummy10, dummy11, dummy12, dummy13 interface{}
		err := indexRows.Scan(
			&dummy1, // Table
			&idx.NonUnique,
			&idx.IndexName,
			&dummy2, // Seq_in_index
			&idx.ColumnName,
			&dummy3,  // Collation
			&dummy4,  // Cardinality
			&dummy5,  // Sub_part
			&dummy6,  // Packed
			&dummy7,  // Null
			&dummy8,  // Index_type
			&dummy9,  // Comment
			&dummy10, // Index_comment
			&dummy11, // Visible
			&dummy12, // Expression
			&dummy13, // Clustered
		)
		if err != nil {
			return info, err
		}
		info.Indexes = append(info.Indexes, idx)
	}

	// 获取CREATE TABLE语句
	var createTable string
	err = db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`", tableName)).Scan(&tableName, &createTable)
	if err != nil {
		return info, err
	}
	info.CreateTable = createTable

	return info, nil
}

func executeExplain(db *sql.DB, sqlQuery string) ([]ExplainResult, error) {
	var results []ExplainResult

	query := strings.TrimSpace(sqlQuery)
	if strings.HasPrefix(strings.ToUpper(query), "EXPLAIN ") {
		query = query[8:]
	}

	rows, err := db.Query(fmt.Sprintf("EXPLAIN %s", query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var res ExplainResult
		var partitions, possibleKeys, key, keyLen, ref, extra sql.NullString
		var filtered sql.NullFloat64

		err := rows.Scan(
			&res.ID,
			&res.SelectType,
			&res.Table,
			&partitions,
			&res.Type,
			&possibleKeys,
			&key,
			&keyLen,
			&ref,
			&res.Rows,
			&filtered,
			&extra,
		)
		if err != nil {
			return nil, err
		}

		res.Partitions = partitions.String
		res.PossibleKeys = possibleKeys.String
		res.Key = key.String
		res.KeyLen = keyLen.String
		res.Ref = ref.String
		res.Filtered = filtered.Float64
		res.Extra = extra.String

		results = append(results, res)
	}

	return results, nil
}

func sendToAI(apiKey string, request AnalysisRequest) (string, error) {
	if apiKey == "" {
		return "未提供OpenAI API key，跳过AI分析", nil
	}

	clientConfig := openai.DefaultConfig(apiKey)
	if config.OpenAIBaseURL != "" {
		baseURL := strings.Trim(config.OpenAIBaseURL, `"`)
		// baseURL = strings.TrimRight(baseURL, "/")
		clientConfig.BaseURL = baseURL
	}
	fmt.Println(clientConfig.BaseURL)
	fmt.Println(apiKey)
	fmt.Println(config.OpenAIModel)

	client := openai.NewClientWithConfig(clientConfig)

	stream, err := client.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: config.OpenAIModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: preparePrompt(request),
				},
			},
			Stream: true,
		},
	)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var fullResponse strings.Builder

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return fullResponse.String(), nil
		}

		if err != nil {
			return "", fmt.Errorf("流式错误: %v", err)
		}

		content := response.Choices[0].Delta.Content
		fullResponse.WriteString(content)
	}
}

func preparePrompt(request AnalysisRequest) string {
	jsonData, _ := json.MarshalIndent(request, "", "  ")

	return fmt.Sprintf(`我有一个MySQL查询分析请求。请分析以下信息并提供：
1. 对EXPLAIN计划结果的解释，特别是每种SelectType和Type为何具有当前值
2. 基于表结构和索引的查询优化建议
3. 当前查询执行计划中可能存在的问题

以下是JSON格式的分析请求数据：

%s

请提供详细的回复，包含清晰的解释和可操作的优化建议。`, jsonData)
}

// 从SQL语句中提取表名
func extractTablesFromSQL(sql string) []string {
	re := regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+([\w.]+)`)
	matches := re.FindAllStringSubmatch(sql, -1)

	tables := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			seen[match[1]] = true
			tables = append(tables, match[1])
		}
	}

	return tables
}
