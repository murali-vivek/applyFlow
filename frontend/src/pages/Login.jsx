import { googleLoginUrl } from '../api';

export default function Login() {
  return (
    <div className="login-page">
      <div className="card login-card">
        <h1>ApplyFlow</h1>
        <p className="login-tagline">Your premium career launchpad — automate thoughtful outreach to recruiters.</p>
        <a href={googleLoginUrl()} className="btn btn-primary" style={{ marginTop: 20, display: 'inline-block' }}>
          Sign in with Google
        </a>
      </div>
    </div>
  );
}
