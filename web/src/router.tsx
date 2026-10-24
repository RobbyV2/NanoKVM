import { createHashRouter, Outlet } from 'react-router-dom';

import { PermissionRoute, ProtectedRoute } from '@/components/auth';
import { RouteError } from '@/components/error-boundary';
import { Root } from '@/components/root';

// Every route carries an errorElement. Without one, react-router walks up to its
// own default handler and replaces the entire page with a bare "Unexpected
// Application Error!" - and because the failure is above our own boundaries,
// nothing on the page survives to explain it or to offer a way back.
export const router = createHashRouter([
  {
    path: '/auth/login',
    errorElement: <RouteError />,
    lazy: async () => {
      const { Login } = await import('./pages/auth/login');
      return { Component: Login };
    }
  },
  {
    path: '/',
    errorElement: <RouteError />,
    element: (
      <ProtectedRoute>
        <Root />
      </ProtectedRoute>
    ),
    children: [
      {
        path: '',
        errorElement: <RouteError />,
        lazy: async () => {
          const { Desktop } = await import('./pages/desktop');
          return { Component: Desktop };
        }
      },
      {
        errorElement: <RouteError />,
        element: (
          <PermissionRoute permission="terminal">
            <Outlet />
          </PermissionRoute>
        ),
        children: [
          {
            path: 'terminal',
            errorElement: <RouteError />,
            lazy: async () => {
              const { Terminal } = await import('./pages/terminal');
              return { Component: Terminal };
            }
          }
        ]
      },
      {
        errorElement: <RouteError />,
        element: (
          <PermissionRoute permission="library.read">
            <Outlet />
          </PermissionRoute>
        ),
        children: [
          {
            path: 'storage',
            errorElement: <RouteError />,
            lazy: async () => {
              const { Storage } = await import('./pages/storage');
              return { Component: Storage };
            }
          }
        ]
      },
      {
        path: 'auth/password',
        errorElement: <RouteError />,
        lazy: async () => {
          const { Password } = await import('./pages/auth/password');
          return { Component: Password };
        }
      }
    ]
  },
  {
    path: '/wifi',
    caseSensitive: false,
    errorElement: <RouteError />,
    lazy: async () => {
      const { Wifi } = await import('./pages/wifi');
      return { Component: Wifi };
    }
  }
]);
