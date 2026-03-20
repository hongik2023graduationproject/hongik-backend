package service

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
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

	// 임시 파일에 코드 작성 후 파일 모드로 실행
	tmpFile, err := os.CreateTemp("", "hongik-*.hik")
	if err != nil {
		return model.ExecuteResponse{
			Status: "error",
			Error:  "임시 파일 생성 실패: " + err.Error(),
		}
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(req.Code); err != nil {
		tmpFile.Close()
		return model.ExecuteResponse{
			Status: "error",
			Error:  "코드 저장 실패: " + err.Error(),
		}
	}
	tmpFile.Close()

	start := time.Now()

	cmd := exec.CommandContext(ctx, s.cfg.InterpreterPath, tmpFile.Name())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	elapsed := time.Since(start).Milliseconds()

	if ctx.Err() == context.DeadlineExceeded {
		return model.ExecuteResponse{
			Status:          "timeout",
			Error:           fmt.Sprintf("실행 시간 초과 (%d초)", timeout),
			ExecutionTimeMs: elapsed,
		}
	}

	output := stdout.String()

	// Truncate output if it exceeds the max output size
	maxBytes := s.cfg.MaxOutputBytes
	if maxBytes > 0 && len(output) > maxBytes {
		output = output[:maxBytes] + fmt.Sprintf("\n\n... 출력이 %d바이트 제한을 초과하여 잘렸습니다", maxBytes)
	}

	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return model.ExecuteResponse{
			Status:          "error",
			Output:          output,
			Error:           errMsg,
			ExecutionTimeMs: elapsed,
		}
	}

	return model.ExecuteResponse{
		Status:          "success",
		Output:          output,
		ExecutionTimeMs: elapsed,
	}
}
