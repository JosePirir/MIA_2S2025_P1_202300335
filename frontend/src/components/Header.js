import React from 'react';
import { Link, useNavigate } from 'react-router-dom';

function Header({ onExecute }) {
  const navigate = useNavigate();

  const handleLogout = async () => {
    try {
      const result = await onExecute('logout');
      alert(result || 'Sesión cerrada');
      navigate('/login');
    } catch (err) {
      alert('Error al cerrar sesión: ' + (err.message || err));
    }
  };

  return (
    <header className="py-3 border-bottom mb-3">
      <div className="container d-flex justify-content-between align-items-center">
        <Link to="/" className="text-decoration-none text-light">
          <h1 className="h4 mb-0">Mi FS - Interfaz</h1>
        </Link>
        <div>
          <Link to="/login" className="btn btn-outline-light me-2">Iniciar sesión</Link>
          <Link to="/discos" className="btn btn-outline-light me-2">Discos</Link>
          <button onClick={handleLogout} className="btn btn-outline-danger">Logout</button>
        </div>
      </div>
    </header>
  );
}

export default Header;
