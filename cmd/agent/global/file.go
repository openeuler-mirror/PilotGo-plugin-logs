/*
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
package global

import (
	"io"
	"os"

	"github.com/pkg/errors"
)

func FileReadString(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", errors.New(err.Error())
	}

	return string(content), nil
}

func FileReadBytes(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer f.Close()

	var content []byte
	readbuff := make([]byte, 1024*4)
	for {
		n, err := f.Read(readbuff)
		if err != nil {
			if err == io.EOF {
				if n != 0 {
					content = append(content, readbuff[:n]...)
				}
				break
			}
			return nil, errors.New(err.Error())
		}
		content = append(content, readbuff[:n]...)
	}

	return content, nil
}
