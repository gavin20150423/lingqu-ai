/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // 主色调 - Comic red
        primary: {
          50: '#fff1ef',
          100: '#ffe1dc',
          200: '#ffc5bb',
          300: '#ff9b8c',
          400: '#ff6e5c',
          500: '#f04438',
          600: '#d92d20',
          700: '#b42318',
          800: '#912018',
          900: '#7a271a',
          950: '#45100b'
        },
        // 辅助色 - comic cyan
        accent: {
          50: '#eefbff',
          100: '#d6f5ff',
          200: '#acecff',
          300: '#72ddff',
          400: '#2bc5f4',
          500: '#08a9d6',
          600: '#0788b5',
          700: '#0b6d92',
          800: '#115977',
          900: '#134a64',
          950: '#082f44'
        },
        comic: {
          ink: '#211f1c',
          paper: '#fffdf5',
          panel: '#fff9dc',
          yellow: '#ffd447',
          red: '#f04438',
          cyan: '#08a9d6',
          mint: '#2ecf9f',
          violet: '#7c5cff'
        },
        // 深色模式背景 - ink scale
        dark: {
          50: '#fbfaf6',
          100: '#ece7da',
          200: '#d3ccb9',
          300: '#b6ac93',
          400: '#92866d',
          500: '#736850',
          600: '#5c523f',
          700: '#433b30',
          800: '#2d2924',
          900: '#211f1c',
          950: '#12110f'
        }
      },
      fontFamily: {
        sans: [
          'Avenir Next',
          'Trebuchet MS',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'system-ui',
          'sans-serif'
        ],
        display: ['Marker Felt', 'Comic Sans MS', 'Trebuchet MS', 'PingFang SC', 'sans-serif'],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: '5px 5px 0 rgba(33, 31, 28, 0.95)',
        'glass-sm': '3px 3px 0 rgba(33, 31, 28, 0.9)',
        glow: '0 0 0 3px rgba(255, 212, 71, 0.45), 5px 5px 0 rgba(33, 31, 28, 0.95)',
        'glow-lg': '0 0 0 5px rgba(255, 212, 71, 0.42), 8px 8px 0 rgba(33, 31, 28, 0.95)',
        card: '4px 4px 0 rgba(33, 31, 28, 0.95)',
        'card-hover': '7px 7px 0 rgba(33, 31, 28, 0.95)',
        'inner-glow': 'inset 0 2px 0 rgba(255, 255, 255, 0.75)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #f04438 0%, #ff8a3d 100%)',
        'gradient-dark': 'linear-gradient(135deg, #211f1c 0%, #12110f 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,253,245,0.9) 0%, rgba(255,249,220,0.78) 100%)',
        'mesh-gradient':
          'radial-gradient(circle at 12px 12px, rgba(33, 31, 28, 0.12) 2px, transparent 2.5px), linear-gradient(135deg, rgba(255, 212, 71, 0.18), rgba(8, 169, 214, 0.14), rgba(240, 68, 56, 0.12))'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 0 3px rgba(255, 212, 71, 0.28), 4px 4px 0 rgba(33, 31, 28, 0.9)' },
          '100%': { boxShadow: '0 0 0 5px rgba(8, 169, 214, 0.24), 7px 7px 0 rgba(33, 31, 28, 0.95)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
