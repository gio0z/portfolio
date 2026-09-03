import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { AdminLayout } from './AdminLayout';
import { adminApi } from './AdminApi';

describe('AdminLayout', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders admin navigation, loads user session, and handles logout', async () => {
    vi.spyOn(adminApi, 'getSession').mockResolvedValue({
      authenticated: true,
      login: 'gio0z',
      csrf_token: 'csrf-xyz',
    });
    const logoutSpy = vi.spyOn(adminApi, 'logout').mockResolvedValue();

    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <AdminLayout />
      </MemoryRouter>
    );

    // Assert main heading matches review queue
    expect(screen.getByRole('heading', { name: /review queue/i })).toBeInTheDocument();

    // Assert navigation links
    expect(screen.getAllByRole('link', { name: /overview/i }).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole('link', { name: /review queue/i }).length).toBeGreaterThanOrEqual(1);

    // Assert user session is displayed
    await waitFor(() => {
      expect(screen.getByText(/gio0z/i)).toBeInTheDocument();
    });

    // Assert logout action
    const logoutBtn = screen.getByRole('button', { name: /log\s*out|sign\s*out/i });
    await user.click(logoutBtn);

    expect(logoutSpy).toHaveBeenCalled();
  });

  it('handles unauthenticated state gracefully', async () => {
    vi.spyOn(adminApi, 'getSession').mockResolvedValue({
      authenticated: false,
    });

    render(
      <MemoryRouter>
        <AdminLayout />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/not signed in/i)).toBeInTheDocument();
    });
    expect(screen.getByRole('link', { name: /sign in/i })).toHaveAttribute(
      'href',
      '/api/admin/auth/login'
    );
  });
});
