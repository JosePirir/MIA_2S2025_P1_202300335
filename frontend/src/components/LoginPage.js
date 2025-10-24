import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';

function LoginPage({ onExecute }) {
  const [user, setUser] = useState('');
  const [pass, setPass] = useState('');
  const [id, setId] = useState('');
  const [loading, setLoading] = useState(false);
  const firstInput = useRef(null);
  const navigate = useNavigate();

  useEffect(() => {
    setUser('');
    setPass('');
    setId('');
    setTimeout(() => firstInput.current && firstInput.current.focus(), 50);
  }, []);

  const submit = async (e) => {
    e.preventDefault();
    if (!user || !pass || !id) {
      alert('Completa usuario, contraseña e id.');
      return;
    }
    setLoading(true);
    try {
      const cmd = `login -user=${user} -pass=${pass} -id=${id}`;
      const result = await onExecute(cmd); // ahora onExecute devuelve la salida
      if (typeof result === 'string' && result.includes('Login exitoso para el usuario')) {
        navigate('/');
        return;
      }
      alert('Login fallido:\n' + (result || 'Sin respuesta'));
    } catch (err) {
      alert('Error: ' + (err.message || err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="container mt-4">
      <div className="card bg-dark text-light mx-auto" style={{ maxWidth: 480 }}>
        <div className="card-header">
          <h5 className="mb-0">Iniciar sesión</h5>
        </div>
        <form onSubmit={submit}>
          <div className="card-body">
            <div className="mb-2">
              <label className="form-label">Usuario</label>
              <input ref={firstInput} value={user} onChange={e => setUser(e.target.value)} className="form-control" />
            </div>
            <div className="mb-2">
              <label className="form-label">Contraseña</label>
              <input type="password" value={pass} onChange={e => setPass(e.target.value)} className="form-control" />
            </div>
            <div className="mb-2">
              <label className="form-label">ID Partición</label>
              <input value={id} onChange={e => setId(e.target.value)} className="form-control" placeholder="e.g. vda1" />
            </div>
          </div>
          <div className="card-footer d-flex justify-content-end">
            <button type="button" className="btn btn-secondary me-2" onClick={() => navigate('/') } disabled={loading}>Cancelar</button>
            <button type="submit" className="btn btn-primary" disabled={loading}>{loading ? 'Conectando...' : 'Entrar'}</button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default LoginPage;