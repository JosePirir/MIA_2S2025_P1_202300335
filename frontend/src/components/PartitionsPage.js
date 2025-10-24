import React, { useEffect, useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';

function useQuery() {
  return new URLSearchParams(useLocation().search);
}

function PartitionsPage({ onExecute }) {
  const [parts, setParts] = useState([]);
  const [loading, setLoading] = useState(false);
  const query = useQuery();
  const navigate = useNavigate();
  const path = query.get('path') || '';

  useEffect(() => {
    if (!path) return;
    load();
    // eslint-disable-next-line
  }, [path]);

  async function load() {
    setLoading(true);
    try {
      // No usar comillas en el flag (el backend acepta / en el valor)
      const res = await onExecute(`listpartitions -path=${decodeURIComponent(path)}`);
      const lines = (res || '').split('\n').map(l => l.trim()).filter(Boolean);
      // Cada línea: TYPE|NAME|START|SIZE|STATUS
      const parsed = lines.map(line => {
        const parts = line.split('|');
        return {
          type: parts[0] || '',
          name: parts[1] || '',
          start: parts[2] || '',
          size: parts[3] || '',
          status: parts[4] || '',
        };
      });
      setParts(parsed);
    } catch (err) {
      setParts([{ type: 'ERROR', name: err.message || String(err), start: '', size: '', status: '' }]);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="container mt-4">
      <div className="d-flex justify-content-between align-items-center mb-3">
        <h4>Particiones del disco</h4>
        <div>
          <button className="btn btn-secondary me-2" onClick={() => navigate(-1)}>Volver</button>
        </div>
      </div>

      <p><strong>Disco:</strong> {decodeURIComponent(path)}</p>

      {loading && <p>Cargando particiones...</p>}

      <table className="table table-dark table-striped">
        <thead>
          <tr>
            <th>Tipo</th>
            <th>Nombre</th>
            <th>Inicio</th>
            <th>Tamaño</th>
            <th>Estado</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          {parts.map((p, i) => (
            <tr key={i}>
              <td>{p.type}</td>
              <td style={{ wordBreak: 'break-all' }}>{p.name}</td>
              <td>{p.start}</td>
              <td>{p.size}</td>
              <td>{p.status}</td>
              <td>
                <Link to={`/browse?disk=${encodeURIComponent(decodeURIComponent(path))}&start=${encodeURIComponent(p.start)}&path=/`} className="btn btn-sm btn-outline-light">
                  Navegar
                </Link>
              </td>
            </tr>
          ))}
          {(!loading && parts.length === 0) && <tr><td colSpan="6">No se encontraron particiones.</td></tr>}
        </tbody>
      </table>
    </div>
  );
}

export default PartitionsPage;