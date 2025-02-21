package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sashabaranov/go-openai"
)

func main() {
	cfg := openai.DefaultConfig(os.Getenv("LLM_API_KEY"))
	baseUrl := os.Getenv("LLM_BASE_URL")
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "deepseek-r1"
	}
	if baseUrl == "" {
		//cfg.BaseURL = "https://api.deepseek.com/v1"
		cfg.BaseURL = "https://api.lkeap.cloud.tencent.com/v1"
	} else {
		cfg.BaseURL = baseUrl
	}
	client := openai.NewClientWithConfig(cfg)

	req := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "you are a helpful chatbot",
			},
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "使用简体中文回答，符合简体中文的表达习惯",
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
		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: s.Text(),
		})
		stream, err := client.CreateChatCompletionStream(context.Background(), req)
		if err != nil {
			fmt.Printf("ChatCompletionStream error: %v\n", err)
			continue
		}
		isDone := false
		isAnswering := false
		fmt.Println("====thinking===")
		for !isDone {
			resp, streamErr := stream.Recv()
			if errors.Is(streamErr, io.EOF) {
				break
			}
			if len(resp.Choices) == 0 {
				// should be include Usage
				if resp.Usage != nil {
					fmt.Printf("PromptTokens: %d, CompletionTokens: %di, Total: %d\n",
						resp.Usage.PromptTokens, resp.Usage.CompletionTokens,
						resp.Usage.TotalTokens)
				}
			}
			if resp.Choices[0].Delta.Content == "" &&
				resp.Choices[0].Delta.ReasoningContent == "" {
				// process blank
				continue
			}
			if resp.Choices[0].Delta.ReasoningContent == "" && !isAnswering {
				fmt.Printf("\n====result content===\n")
				isAnswering = true
			}
			if resp.Choices[0].Delta.ReasoningContent != "" {
				fmt.Printf("%s", resp.Choices[0].Delta.ReasoningContent)
			}
			if resp.Choices[0].Delta.Content != "" {
				fmt.Printf("%s", resp.Choices[0].Delta.Content)
			}
		}
		stream.Close()
		fmt.Print("> ")
	}
}
