import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { setToken } from '../api';

export default function AuthCallback() {
  const [params] = useSearchParams();
  const navigate = useNavigate();

  useEffect(() => {
    const token = params.get('token');
    if (token) {
      setToken(token);
      navigate('/dashboard', { replace: true });
    } else {
      navigate('/', { replace: true });
    }
  }, [params, navigate]);

  return <div className="layout">Signing in...</div>;
}
