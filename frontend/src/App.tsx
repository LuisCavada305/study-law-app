import { NavLink, Outlet } from 'react-router-dom'

export default function App() {
  return (
    <>
      <header className="topo">
        <NavLink to="/" className="marca">
          <span className="marca-simbolo">§</span> Lei Seca
        </NavLink>
        <nav>
          <NavLink to="/" end>
            Leis
          </NavLink>
          <NavLink to="/estudo">Estudar</NavLink>
        </nav>
      </header>
      <main>
        <Outlet />
      </main>
    </>
  )
}
