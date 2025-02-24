package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sashabaranov/go-openai"
)

var (
	platName  string
	apiKey    string
	modelName string
	baseUrl   string
	bVerbose  bool
)

type platFormType struct {
	baseUrl   string
	modelName string
}

// "gemini-2.0-flash-thinking-exp-01-21"
var platforms = map[string]platFormType{
	"tencent": {"https://api.lkeap.cloud.tencent.com/v1", "deepseek-r1"},
	"aliyun":  {"https://dashscope.aliyuncs.com/compatible-mode/v1", "deepseek-r1"},
	"groq":    {"https://api.groq.com/openai/v1", "llama-3.3-70b-versatile"},
	"gemini": {"https://generativelanguage.googleapis.com/v1beta/openai",
		"gemini-2.0-flash-thinking-exp"},
	"siliconflow": {"https://api.siliconflow.cn/v1", "deepseek-ai/DeepSeek-R1"},
	"deepseek":    {"https://api.deepseek.com/v1", "deepseek-reasoner"},
}

func main() {
	var bList bool
	flag.StringVar(&platName, "plat", "", "platform name")
	flag.StringVar(&apiKey, "apikey", "", "LLM API_KEY")
	flag.StringVar(&modelName, "model", "", "LLM model name")
	flag.StringVar(&baseUrl, "baseURI", "", "openai API baseURI")
	flag.BoolVar(&bVerbose, "v", false, "verbose log")
	flag.BoolVar(&bList, "list", false, "list models")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: auction [options]\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()

	if platName != "" {
		if plat, ok := platforms[platName]; ok {
			if baseUrl == "" {
				baseUrl = plat.baseUrl
			}
			if modelName == "" {
				modelName = plat.modelName
			}
		}
	}
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	cfg := openai.DefaultConfig(apiKey)
	if baseUrl == "" {
		baseUrl = os.Getenv("LLM_BASE_URL")
	}
	if modelName == "" {
		modelName = os.Getenv("LLM_MODEL")
		if modelName == "" {
			modelName = "deepseek-r1"
		}
	}
	if baseUrl == "" {
		//cfg.BaseURL = "https://api.deepseek.com/v1"
		cfg.BaseURL = "https://api.lkeap.cloud.tencent.com/v1"
	} else {
		cfg.BaseURL = baseUrl
	}
	client := openai.NewClientWithConfig(cfg)
	if bList {
		modelList, err := client.ListModels(context.Background())
		if err != nil {
			fmt.Println("ListModels error:", err)
			return
		}
		for _, modelA := range modelList.Models {
			fmt.Printf("%s (%s) created %d owner(%s) Window(%d)\n",
				modelA.ID, modelA.Object, modelA.CreatedAt,
				modelA.OwnedBy, modelA.ContextWindow)
		}
		return
	}

	req := openai.ChatCompletionRequest{
		Model: modelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "you are a helpful chatbot",
			},
		},
		Stream: true,
	}
	fmt.Println("Conversation. Enter /bye to exit")
	fmt.Println("---------------------")
	fmt.Print("> ")
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		if s.Text() == "/bye" {
			break
		}
		if s.Text() == "" {
			continue
		}
		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: s.Text(),
		})
		stream, err := client.CreateChatCompletionStream(context.Background(), req)
		if err != nil {
			fmt.Printf("ChatCompletionStream error: %v\n", err)
			break
		}
		isDone := false
		isAnswering := false
		hasReasoning := false
		var usage *openai.Usage
		for !isDone {
			resp, streamErr := stream.Recv()
			if errors.Is(streamErr, io.EOF) {
				break
			}
			// should be include Usage
			if resp.Usage != nil {
				usage = resp.Usage
			}
			for _, choice := range resp.Choices {
				if choice.Delta.Content == "" &&
					choice.Delta.ReasoningContent == "" {
					// process blank
					continue
				}
				if choice.Delta.ReasoningContent != "" && !hasReasoning {
					hasReasoning = true
					fmt.Println("====thinking===")
				}
				if choice.Delta.ReasoningContent == "" && !isAnswering {
					if hasReasoning {
						fmt.Printf("\n====result content===\n")
					}
					isAnswering = true
				}
				if choice.Delta.ReasoningContent != "" {
					fmt.Printf("%s", choice.Delta.ReasoningContent)
				}
				if choice.Delta.Content != "" {
					fmt.Printf("%s", choice.Delta.Content)
				}
			}
		}
		if bVerbose && usage != nil {
			fmt.Printf("\n\nPromptTokens: %d, CompletionTokens: %d, Total: %d\n",
				usage.PromptTokens, usage.CompletionTokens,
				usage.TotalTokens)
		} else {
			fmt.Println("")
		}
		stream.Close()
		fmt.Print("> ")
	}
}
