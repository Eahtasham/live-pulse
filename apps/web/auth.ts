import NextAuth from "next-auth"
import Google from "next-auth/providers/google"
import Credentials from "next-auth/providers/credentials"

const apiUrl = process.env.API_URL || "http://localhost:8080"

export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [
    Google,
    Credentials({
      id: "email-login",
      name: "Email Login",
      credentials: {
        email: { label: "Email", type: "email" },
        password: { label: "Password", type: "password" },
      },
      async authorize(credentials) {
        if (!credentials?.email || !credentials?.password) return null

        const res = await fetch(`${apiUrl}/v1/auth/login`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            email: credentials.email,
            password: credentials.password,
          }),
        })

        if (!res.ok) return null
        const data = await res.json()

        return {
          id: credentials.email as string,
          email: credentials.email as string,
          name: credentials.email as string,
          apiToken: data.token,
        }
      },
    }),
  ],
  callbacks: {
    async signIn({ user, account }) {
      if (account?.provider === "google") {
        try {
          const res = await fetch(`${apiUrl}/v1/auth/callback`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              email: user.email,
              name: user.name,
              avatar_url: user.image,
              provider: "google",
            }),
          })
          if (!res.ok) {
            const text = await res.text()
            console.error("[auth] Go API callback failed:", res.status, text)
            return false
          }
          const data = await res.json()
          user.apiToken = data.token
        } catch (err) {
          console.error("[auth] Go API callback error:", err)
          return false
        }
      }
      return true
    },
    async jwt({ token, user }) {
      if (user?.apiToken) {
        token.apiToken = user.apiToken
      }
      return token
    },
    async session({ session, token }) {
      session.apiToken = token.apiToken as string
      return session
    },
  },
  pages: {
    signIn: "/login",
  },
})
