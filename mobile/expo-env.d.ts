declare const process: {
  env: {
    NODE_ENV: 'development' | 'production' | 'test'
    [key: string]: string | undefined
  }
}