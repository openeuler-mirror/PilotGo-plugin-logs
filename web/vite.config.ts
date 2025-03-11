/* 
 * Copyright (c) KylinSoft  Co., Ltd. 2024.All rights reserved.
 * PilotGo-plugin-logs licensed under the Mulan Permissive Software License, Version 2. 
 * See LICENSE file for more details.
 * Author: Wangjunqi123 <wangjunqi@kylinos.cn>
 * Date: Mon Dec 16 08:43:58 2024 +0800
 */
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  base: "/plugin/logs",
  plugins: [vue()],
  server: {
    host:'localhost',
    proxy: {
      '/plugin/logs/api': {
        target: 'https://10.41.107.29:9994',
        secure:false,
        changeOrigin: true,
        rewrite: path => path.replace(/^\//, '')
      },
    },
  }
})
