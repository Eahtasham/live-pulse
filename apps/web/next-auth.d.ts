import "next-auth"
import "next-auth/jwt"

declare module "next-auth" {
  interface User {
    apiToken?: string
  }
  interface Session {
    apiToken?: string
  }
}

declare module "next-auth/jwt" {
  interface JWT {
    apiToken?: string
  }
}
