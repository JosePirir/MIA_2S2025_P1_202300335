import React from 'react';
import { Link } from 'react-router-dom';

function Header() {
  return (
    <header className="py-3 border-bottom mb-3">
      <div className="container d-flex justify-content-between align-items-center">
        <h1 className="h4 text-light mb-0">Mi FS - Interfaz</h1>
        <div>
          <Link to="/login" className="btn btn-outline-light me-2">
            Iniciar sesión
          </Link>
          <Link to="/discos" className="btn btn-outline-light">
            Discos
          </Link>
        </div>
      </div>
    </header>
  );
}

export default Header;