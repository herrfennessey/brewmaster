import { BrowserRouter, Route, Routes } from 'react-router-dom'
import Home from './screens/Home'
import BeanReview from './screens/BeanReview'
import BrewParameters from './screens/BrewParameters'
import RoastDatePrompt from './screens/RoastDatePrompt'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/review/:id" element={<BeanReview />} />
        <Route path="/roast-date/:beanId" element={<RoastDatePrompt />} />
        <Route path="/brew/:beanId" element={<BrewParameters />} />
      </Routes>
    </BrowserRouter>
  )
}
