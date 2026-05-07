import { useEffect } from 'react'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import AppShell from './components/AppShell'
import Home from './screens/Home'
import BeanReview from './screens/BeanReview'
import BrewParameters from './screens/BrewParameters'
import RoastDatePrompt from './screens/RoastDatePrompt'
import SignIn from './screens/SignIn'
import MyCoffees from './screens/MyCoffees'
import CoffeeDetail from './screens/CoffeeDetail'
import ShotDoctor from './screens/ShotDoctor'
import { AuthProvider, useAuth } from './services/auth-context'
import { setIdTokenGetter } from './services/api'

function ApiAuthBridge() {
  const { getIdToken } = useAuth()
  useEffect(() => {
    setIdTokenGetter(getIdToken)
  }, [getIdToken])
  return null
}

export default function App() {
  return (
    <AuthProvider>
      <ApiAuthBridge />
      <BrowserRouter>
        <Routes>
          <Route element={<AppShell />}>
            <Route path="/" element={<Home />} />
            <Route path="/review/:id" element={<BeanReview />} />
            <Route path="/roast-date/:beanId" element={<RoastDatePrompt />} />
            <Route path="/brew/:beanId" element={<BrewParameters />} />
            <Route path="/signin" element={<SignIn />} />
            <Route path="/coffees" element={<MyCoffees />} />
            <Route path="/coffees/:id" element={<CoffeeDetail />} />
            <Route path="/shot-doctor" element={<ShotDoctor />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
