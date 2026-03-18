package service

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"hongik-backend/config"
	"hongik-backend/model"
)

type InterpreterService struct {
	cfg *config.Config
}

func NewInterpreterService(cfg *config.Config) *InterpreterService {
	return &InterpreterService{cfg: cfg}
}

func (s *InterpreterService) Execute(req model.ExecuteRequest) model.ExecuteResponse {
	timeout := s.cfg.ExecuteTimeout
	if req.Timeout >= 1 && req.Timeout <= 30 {
		timeout = req.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	start := time.Now()

	cmd := exec.CommandContext(ctx, s.cfg.InterpreterPath)
	cmd.Stdin = strings.NewReader(req.Code)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	if ctx.Err() == context.DeadlineExceeded {
		return model.ExecuteResponse{
			Status:          "timeout",
			Error:           fmt.Sprintf("실행 시간 초과 (%d초)", timeout),
			ExecutionTimeMs: elapsed,
		}
	}

	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return model.ExecuteResponse{
			Status:          "error",
			Output:          stdout.String(),
			Error:           errMsg,
			ExecutionTimeMs: elapsed,
		}
	}

	return model.ExecuteResponse{
		Status:          "success",
		Output:          stdout.String(),
		ExecutionTimeMs: elapsed,
	}
}
