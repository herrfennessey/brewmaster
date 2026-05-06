import { useEffect } from 'react'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import Home from './screens/Home'
import BeanReview from './screens/BeanReview'
import BrewParameters from './screens/BrewParameters'
import RoastDatePrompt from './screens/RoastDatePrompt'
import SignIn from './screens/SignIn'
import MyCoffees from './screens/MyCoffees'
import CoffeeDetail from './screens/CoffeeDetail'
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
          <Route path="/" element={<Home />} />
          <Route path="/review/:id" element={<BeanReview />} />
          <Route path="/roast-date/:beanId" element={<RoastDatePrompt />} />
          <Route path="/brew/:beanId" element={<BrewParameters />} />
          <Route path="/signin" element={<SignIn />} />
          <Route path="/coffees" element={<MyCoffees />} />
          <Route path="/coffees/:id" element={<CoffeeDetail />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
