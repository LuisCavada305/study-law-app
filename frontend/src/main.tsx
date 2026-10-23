import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
import Estudo from './paginas/Estudo.tsx'
import Leis from './paginas/Leis.tsx'
import Leitura from './paginas/Leitura.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        <Route element={<App />}>
          <Route index element={<Leis />} />
          <Route path="estudo" element={<Estudo />} />
          <Route path="estudo/:leiId" element={<Estudo />} />
          <Route path="leis/:leiId" element={<Leitura />} />
        </Route>
      </Routes>
    </BrowserRouter>
  </StrictMode>,
)
