import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

function DisksPage({ onExecute }) {
  const [disks, setDisks] = useState([]);
  const [loading, setLoading] = useState(false);

  async function load(path = '/home/josepirir/Calificacion_MIA/Discos') {
    setLoading(true);
    try {
      const res = await onExecute(`listdisks -path=${path}`);
      const rawLines = (res || '').split('\n').map(l => l.replace(/\r/g,'').trim()).filter(Boolean);
      const useful = rawLines.filter(l => !l.startsWith('>'));
      setDisks(useful);
    } catch (err) {
      setDisks([`Error: ${err.message || err}`]);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  return (
    <div className="container mt-4">
      <h4>Discos detectados (carpeta "discos")</h4>
      {loading ? <p>Cargando...</p> : null}
      <ul className="list-group">
        {disks.map((d, i) => {
          const hasText = (d || '').trim() !== '';
          return (
            <li key={i} className="list-group-item bg-dark text-light d-flex justify-content-between align-items-center">
              <div style={{ wordBreak: 'break-all' }}>{d}</div>
              {hasText ? (
                <Link to={`/disco?path=${encodeURIComponent(d)}`} className="btn btn-sm btn-outline-light">Ver particiones</Link>
              ) : null}
            </li>
          );
        })}
        {(!loading && disks.length === 0) && <li className="list-group-item bg-dark text-light">No se encontraron discos.</li>}
      </ul>
    </div>
  );
}

export default DisksPage;