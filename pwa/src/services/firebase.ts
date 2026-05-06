import { initializeApp, type FirebaseOptions } from 'firebase/app'
import {
  GoogleAuthProvider,
  getAuth,
  linkWithPopup,
  signInAnonymously,
  signInWithCredential,
  signInWithPopup,
  signOut as firebaseSignOut,
} from 'firebase/auth'

// Firebase web config is non-secret. Values come from VITE_FIREBASE_* envs at
// build time. When unconfigured, init is skipped and the API can run with the
// server's DISABLE_AUTH=true switch.
const firebaseConfig: FirebaseOptions = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY,
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN,
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID,
  appId: import.meta.env.VITE_FIREBASE_APP_ID,
}

export const authBackendConfigured = Boolean(firebaseConfig.apiKey && firebaseConfig.projectId)

const app = authBackendConfigured ? initializeApp(firebaseConfig) : null
export const firebaseAuth = app ? getAuth(app) : null

// ensureAnonymous creates an anonymous user when no one is signed in. We rely
// on this so every device has a stable Firebase UID from first load — it lets
// users save coffees and brew sessions without a sign-in wall, and a later
// link-with-Google preserves all that data under the same UID.
export async function ensureAnonymous(): Promise<void> {
  if (!firebaseAuth || firebaseAuth.currentUser) return
  await signInAnonymously(firebaseAuth)
}

// upgradeToGoogle promotes the current user. When the user is anonymous we
// link the Google credential to their existing UID so all anon-saved data
// follows them. When linking fails because the Google account is already
// attached to another UID (e.g. they signed in on a different device first),
// we fall back to signing in with that credential — losing the anon-only data
// on this device, which is the standard Firebase trade-off.
export async function upgradeToGoogle(): Promise<void> {
  if (!firebaseAuth) throw new Error('Firebase auth is not configured')
  const provider = new GoogleAuthProvider()
  const current = firebaseAuth.currentUser

  if (current?.isAnonymous) {
    try {
      await linkWithPopup(current, provider)
      return
    } catch (err) {
      const code = (err as { code?: string }).code
      // The Google account already belongs to another UID — sign into that
      // account directly. Anon data on this device becomes orphaned.
      if (code === 'auth/credential-already-in-use' || code === 'auth/email-already-in-use') {
        const credential = GoogleAuthProvider.credentialFromError(err as never)
        if (credential) {
          await signInWithCredential(firebaseAuth, credential)
          return
        }
      }
      throw err
    }
  }

  await signInWithPopup(firebaseAuth, provider)
}

export async function googleSignOut(): Promise<void> {
  if (!firebaseAuth) return
  await firebaseSignOut(firebaseAuth)
}
