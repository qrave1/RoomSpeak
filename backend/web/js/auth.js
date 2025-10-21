export async function checkAuth() {
    try {
        const response = await fetch('/api/v1/me');
        if (response.ok) {
            const user = await response.json();
            return {isAuthenticated: true, user: user};
        }
        return {isAuthenticated: false};
    } catch (err) {
        return {isAuthenticated: false};
    }
}

export async function login(auth) {
    try {
        const response = await fetch('/api/auth/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(auth)
        });

        if (response.ok) {
            return true;
        } else {
            alert('Login failed');
            return false;
        }
    } catch (err) {
        console.error('Login error:', err);
        alert('Login error');
        return false;
    }
}

export async function register(auth) {
    try {
        const response = await fetch('/api/auth/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(auth)
        });

        if (response.ok) {
            alert('Registration successful, please login');
            return true;
        } else {
            alert('Registration failed');
            return false;
        }
    } catch (err) {
        console.error('Registration error:', err);
        alert('Registration error');
        return false;
    }
}

export function logout() {
    document.cookie = 'jwt=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
    window.location.reload();
}
