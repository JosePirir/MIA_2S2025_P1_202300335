import React, { useEffect, useState } from 'react';
import { useLocation, useNavigate, Link } from 'react-router-dom';

function useQuery() { return new URLSearchParams(useLocation().search); }

function PartitionsPage({ onExecute }) {
  const [parts, setParts] = useState([]);
  const [loading, setLoading] = useState(false);
  const query = useQuery();
  const navigate = useNavigate();
  const diskPath = query.get('path') || '';

  useEffect(() => { if (diskPath) load(); }, [diskPath]);

  async function load() {
    setLoading(true);
    try {
      const res = await onExecute(`listpartitions -path=${diskPath}`);
      const rawLines = (res || '').split('\n').map(l => l.replace(/\r/g, '').trim()).filter(Boolean);
      const useful = rawLines.filter(l => !l.startsWith('>') && l !== diskPath);
      const parsed = useful.map(line => {
        const fields = line.split('|');
        return {
          type: fields[0] || '',
          name: fields[1] || '',
          start: fields[2] || '',
          size: fields[3] || '',
          status: fields[4] || ''
        };
      });
      setParts(parsed);
    } catch (err) {
      setParts([]);
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

      <p><strong>Disco:</strong> {diskPath}</p>

      {loading && <p>Cargando particiones...</p>}

      <table className="table table-dark table-striped">
        <thead>
          <tr><th>Tipo</th><th>Nombre</th><th>Inicio</th><th>Tamaño</th><th>Estado</th><th>Acciones</th></tr>
        </thead>
        <tbody>
          {parts.map((p, i) => (
            <tr key={i}>
              <td>{p.type}</td>
              <td style={{wordBreak:'break-all'}}>{p.name}</td>
              <td>{p.start}</td>
              <td>{p.size}</td>
              <td>{p.status}</td>
              <td>
                <Link to={`/browse?disk=${encodeURIComponent(diskPath)}&start=${encodeURIComponent(p.start)}&path=/`} className="btn btn-sm btn-outline-light">Navegar</Link>
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