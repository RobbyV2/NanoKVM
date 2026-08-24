const es = {
  translation: {
    head: {
      desktop: 'Escritorio remoto',
      login: 'Inicio de sesión',
      changePassword: 'Cambiar contraseña',
      terminal: 'Consola',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Iniciar sesión',
      placeholderUsername: 'Introduce tu nombre de usuario',
      placeholderPassword: 'Introduce tu contraseña',
      placeholderCurrentPassword: 'Contraseña actual',
      placeholderPassword2: 'Introduce tu contraseña de nuevo',
      noEmptyUsername: 'El nombre de usuario no puede estar vacío',
      noEmptyPassword: 'La contraseña no puede estar vacía',
      passwordLength: 'La contraseña debe tener entre 8 y 72 caracteres',
      noAccount:
        'No se ha encontrado la cuenta. Por favor, recarga la página o recupera tu contraseña.',
      invalidUser: 'Nombre de usuario o contraseña incorrectos',
      locked: 'Demasiados inicios de sesión, inténtalo de nuevo más tarde',
      globalLocked: 'Sistema bajo protección, inténtelo nuevamente más tarde',
      error: 'Error inesperado',
      invalidCurrentPassword: 'La contraseña actual es incorrecta',
      changePassword: 'Cambiar contraseña',
      changePasswordDesc:
        'Para la seguridad de su dispositivo, por favor, modifique la contraseña de inicio de sesión en la web.',
      differentPassword: 'Las contraseñas no coinciden',
      illegalUsername: 'El  nombre de usuario contiene caracteres no permitidos',
      illegalPassword: 'La contraseña contiene caracteres no permitidos',
      forgetPassword: 'Contraseña olvidada',
      ok: 'Aceptar',
      cancel: 'Cancelar',
      loginButtonText: 'Iniciar sesión',
      tips: {
        reset1:
          'Para restablecer las contraseñas, mantén pulsado el botón BOOT del NanoKVM durante 10 segundos.',
        reset2: 'Para ver los pasos detallados, consulta este documento:',
        reset3: 'Cuenta predeterminada de la interfaz web:',
        reset4: 'Cuenta predeterminada de SSH:',
        change1: 'Ten en cuenta que esta acción cambiará las siguientes contraseñas:',
        change2: 'Contraseña de acceso web',
        change3: 'Contraseña root del sistema (contraseña de acceso por SSH)',
        change4: 'Para restablecer las contraseñas, mantén pulsado el botón BOOT del NanoKVM.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Configura el Wi-Fi para el NanoKVM',
      success: 'Comprueba el estado de red del NanoKVM y accede a la nueva dirección IP.',
      failed: 'La operación ha fallado, vuelve a intentarlo.',
      invalidMode:
        'El modo actual no admite la configuración de red. Vaya a su dispositivo y habilite el modo de configuración Wi-Fi.',
      confirmBtn: 'Aceptar',
      finishBtn: 'Finalizado',
      ap: {
        authTitle: 'Autenticación requerida',
        authDescription: 'Por favor ingrese la contraseña AP para continuar',
        authFailed: 'Contraseña AP no válida',
        passPlaceholder: 'AP contraseña',
        verifyBtn: 'Verificar'
      }
    },
    screen: {
      scale: 'Escala',
      title: 'Pantalla',
      video: 'Modo de vídeo',
      videoDirectTips: 'Habilita HTTPS en "Ajustes > Dispositivo" para usar este modo',
      resolution: 'Resolución',
      auto: 'Automático',
      autoTips:
        'En determinadas resoluciones pueden producirse rasgado de imagen (tearing) o desplazamiento del ratón. Prueba a ajustar la resolución del host remoto o desactiva el modo automático.',
      fps: 'FPS',
      customizeFps: 'Personalizar',
      quality: 'Calidad',
      qualityLossless: 'Sin pérdida',
      qualityHigh: 'Alto',
      qualityMedium: 'Medio',
      qualityLow: 'Bajo',
      frameDetect: 'Detectar fotogramas',
      frameDetectTip:
        'Calcula la diferencia entre fotogramas. Para de transmitir vídeo cuando no se detectan cambios en la pantalla del host remoto.',
      resetHdmi: 'Reiniciar HDMI',
      mixedH264: {
        title: 'Conflicto de flujo H.264',
        description:
          'H.264 Direct y H.264 WebRTC se están utilizando al mismo tiempo. Esto puede causar tearing de pantalla o vídeo corrupto. Utilice solo un modo H.264.'
      },
      webrtcConnectionFailed: {
        title: 'Error de conexión de WebRTC',
        description: 'Compruebe la conexión de red o cambie el modo de vídeo.'
      },
      captureStatus: {
        hdmiError: 'Error de imagen HDMI',
        unsupportedResolution: 'La resolución actual no es compatible',
        retrieving: 'Obteniendo pantalla...',
        changingResolution: 'Cambiando resolución...',
        updateFailed: 'La pantalla no puede actualizarse ahora',
        videoError: 'Error de visualización de video',
        noHdmi: 'No se detectó señal HDMI',
        unavailable: 'La pantalla no puede mostrarse ahora'
      }
    },
    keyboard: {
      title: 'Teclado',
      paste: 'Pegar',
      tips: 'Sólo están soportadas las letras y símbolos estándar del teclado',
      placeholder: 'Por favor, introduce el texto',
      submit: 'Enviar',
      virtual: 'Teclado virtual',
      readClipboard: 'Leer del portapapeles',
      clipboardPermissionDenied:
        'Permiso de portapapeles denegado. Por favor, permite el acceso al portapapeles en tu navegador.',
      clipboardReadError: 'Error al leer del portapapeles',
      dropdownEnglish: 'Inglés',
      dropdownGerman: 'Alemán',
      dropdownFrench: 'Francés',
      dropdownRussian: 'ruso',
      shortcut: {
        title: 'Atajos',
        custom: 'Personalizado',
        capture: 'Haga clic aquí para capturar el acceso directo',
        clear: 'Borrar',
        save: 'Guardar',
        captureTips:
          'Capturar teclas del sistema (como la tecla Windows) requiere permiso de pantalla completa.',
        enterFullScreen: 'Alternar el modo de pantalla completa.'
      },
      leaderKey: {
        title: 'Tecla líder',
        desc: 'Omite las restricciones del navegador y envía accesos directos al sistema directamente al host remoto.',
        howToUse: 'Cómo utilizar',
        simultaneous: {
          title: 'Modo Simultáneo',
          desc1: 'Mantenga pulsada la tecla líder y luego pulse el atajo.',
          desc2: 'Intuitivo, pero puede entrar en conflicto con los atajos del sistema.'
        },
        sequential: {
          title: 'Modo Secuencial',
          desc1:
            'Pulse la tecla líder → pulse el atajo en secuencia → vuelva a pulsar la tecla líder.',
          desc2: 'Requiere más pasos, pero evita por completo conflictos del sistema.'
        },
        enable: 'Habilitar tecla líder',
        tip: 'Al asignarse como tecla líder, esta tecla funciona únicamente como activador de atajos y pierde su comportamiento predeterminado.',
        placeholder: 'Pulse la tecla líder',
        shiftRight: 'Shift derecho',
        ctrlRight: 'Ctrl derecho',
        metaRight: 'Win derecho',
        submit: 'Enviar',
        recorder: {
          rec: 'REC',
          activate: 'Activar teclas',
          input: 'Por favor presione el atajo...'
        }
      }
    },
    mouse: {
      title: 'Ratón',
      cursor: 'Estilo de cursor',
      default: 'Cursor por defecto',
      pointer: 'Cursor de puntero',
      cell: 'Cursor de celda',
      text: 'Cursor de texto',
      grab: 'Cursor de agarre',
      hide: 'Ocultar cursor',
      mode: 'Modo de ratón',
      absolute: 'Modo absoluto',
      relative: 'Modo relativo',
      direction: 'Dirección de la rueda de desplazamiento',
      scrollUp: 'Desplazarse hacia arriba',
      scrollDown: 'Desplácese hacia abajo',
      speed: 'Velocidad de la rueda de desplazamiento',
      fast: 'Rápida',
      slow: 'Lenta',
      requestPointer:
        'Usando modo relativo. Por favor, haz clic en el escritorio para obtener el cursor del ratón.',
      resetHid: 'Restablecer HID',
      hidOnly: {
        title: 'Modo solo HID',
        desc: 'Si tu ratón y teclado dejan de responder y restablecer el HID no ayuda, podría ser un problema de compatibilidad entre el NanoKVM y el dispositivo. Prueba a habilitar el modo sólo HID para mejorar la compatibilidad.',
        tip1: 'Habilitar el modo sólo HID desmontará el disco virtual y la red virtual',
        tip2: 'En modo sólo HID, el montaje de imágenes está deshabilitado',
        tip3: 'El NanoKVM se reiniciará automáticamente después de cambiar de modo',
        enable: 'Habilitar modo sólo HID',
        disable: 'Desactivar modo sólo HID'
      }
    },
    image: {
      title: 'Imágenes',
      loading: 'Cargando...',
      empty: 'No se ha encontrado nada',
      mountMode: 'Modo de montaje',
      mountFailed: 'Fallo al montar',
      mountDesc:
        'En algunos sistemas, es necesario expulsar el disco virtual del host remoto antes de montar una imagen.',
      unmountFailed: 'Fallo al desmontar',
      unmountDesc:
        'En algunos sistemas, es necesario expulsar manualmente el disco virtual desde el host remoto antes de desmontar la imagen.',
      refresh: 'Actualizar la lista de imágenes',
      attention: 'Atención',
      deleteConfirm: '¿Estás seguro de que deseas eliminar esta imagen?',
      okBtn: 'Sí',
      cancelBtn: 'No',
      tips: {
        title: 'Cómo subir imágenes',
        usb1: 'Conecta el NanoKVM a tu computadora mediante USB.',
        usb2: 'Asegúrate de que el disco virtual esté montado (Ajustes - Disco Virtual).',
        usb3: 'Abre el disco virtual en tu computadora y copia el archivo de imagen en el directorio raíz del disco virtual.',
        scp1: 'Asegúrate de que el NanoKVM y tu computadora estén en la misma red local.',
        scp2: 'Abre una terminal en tu computadora y usa el comando SCP para subir el archivo de imagen al directorio /data en el NanoKVM.',
        scp3: 'Ejemplo: scp tu-ruta-de-imagen root@tu-ip-del-nanokvm:/data',
        tfCard: 'Tarjeta SD',
        tf1: 'Este método es compatible con el sistema Linux',
        tf2: 'Obtén la tarjeta SD del NanoKVM (para la versión FULL, desmonta la carcasa primero).',
        tf3: 'Inserta la tarjeta SD en un lector de tarjetas y conéctalo a tu computadora.',
        tf4: 'Copia el archivo de imagen en el directorio /data de la tarjeta SD.',
        tf5: 'Inserta la tarjeta SD en el NanoKVM.'
      }
    },
    script: {
      title: 'Script',
      upload: 'Subir',
      run: 'Ejecutar',
      runBackground: 'Ejecutar en segundo plano',
      runFailed: 'Ejecución fallida',
      attention: 'Atención',
      delDesc: '¿Estás seguro de que deseas eliminar este archivo?',
      confirm: 'Sí',
      cancel: 'No',
      delete: 'Eliminar',
      close: 'Cerrar'
    },
    terminal: {
      title: 'Consola',
      nanokvm: 'Consola del NanoKVM',
      serial: 'Consola del Puerto Serie',
      serialPort: 'Puerto Serie',
      serialPortPlaceholder: 'Por favor, introduce el puerto serie',
      baudrate: 'Tasa de baudios',
      parity: 'Paridad',
      parityNone: 'Ninguna',
      parityEven: 'Par',
      parityOdd: 'Impar',
      flowControl: 'Control de flujo',
      flowControlNone: 'Ninguno',
      flowControlSoft: 'Software',
      flowControlHard: 'Hardware',
      dataBits: 'Bits de datos',
      stopBits: 'Bits de parada',
      confirm: 'Confirmar'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Enviando comando...',
      sent: 'Comando enviado',
      input: 'Por favor, introduce la dirección MAC',
      ok: 'Aceptar'
    },
    download: {
      title: 'Descargador de imágenes',
      input: 'Por favor, introduce la URL de una imagen remota',
      ok: 'Aceptar',
      disabled: 'La partición /data es de sólo lectura, no se puede descargar la imagen',
      uploadbox: 'Suelte el archivo aquí o haga clic para seleccionar',
      inputfile: 'Por favor ingrese el archivo de imagen',
      NoISO: 'Sin ISO',
      sha256: 'SHA-256 (opcional)',
      sha256Placeholder: 'Introduzca una suma de comprobación SHA-256 de 64 caracteres',
      invalidSHA256: 'SHA-256 debe ser una cadena hexadecimal de 64 caracteres',
      failed: 'Descarga fallida',
      success: 'Descarga correcta',
      checksumFailed: 'Descarga fallida: error en la verificación SHA-256',
      cancel: 'Cancelar',
      cancelFailed: 'No se pudo cancelar la descarga'
    },
    power: {
      title: 'Encender / Apagar',
      showConfirm: 'Confirmación',
      showConfirmTip: 'Las operaciones de encendido requieren confirmación adicional',
      reset: 'Reiniciar',
      power: 'Encender / Apagar',
      powerShort: 'Encender / Apagar (pulsación corta)',
      powerLong: 'Encendido/Apagado (pulsación larga)',
      resetConfirm: '¿Desea proceder con la operación de reinicio?',
      powerConfirm: '¿Desea proceder con la operación de encendido?',
      okBtn: 'Sí',
      cancelBtn: 'No'
    },
    devices: {
      title: 'Dispositivos',
      stale: 'El estado en vivo de los dispositivos no está disponible. Reconectando.',
      empty: 'No hay ninguna ranura de cámara o micrófono configurada.',
      available: 'Disponible',
      waiting: 'El host espera una fuente',
      hostOpen: 'Host abierto',
      hostIdle: 'Host inactivo',
      sending: 'Enviando desde este navegador',
      black: 'Vídeo en negro',
      silence: 'Silencio digital',
      resuming: 'Esperando para reanudar',
      stop: 'Dejar de compartir',
      disconnect: 'Desconectar',
      takeover: 'Tomar el control',
      refused: 'En uso por {{owner}} desde {{source}}',
      connectedSources_one: '{{count}} fuente conectada',
      connectedSources_other: '{{count}} fuentes conectadas',
      connectedSources: '{{count}} fuentes conectadas',
      connection: {
        connecting: 'Conectando',
        connected: 'En vivo',
        disconnected: 'Reconectando'
      },
      share: {
        camera: 'Compartir cámara',
        microphone: 'Compartir micrófono',
        usbDevice: 'Compartir USB'
      },
      permission: {
        denied: 'Bloqueado en la configuración del sitio de tu navegador',
        prompt: 'El navegador pedirá permiso'
      },
      mic: {
        mute: 'Silenciar',
        unmute: 'Dejar de silenciar'
      },
      revoked: {
        released: 'Se detuvo la compartición',
        lease_expired: 'La concesión caducó antes de que este navegador volviera',
        admin_disconnect: 'Un administrador desconectó todas las fuentes',
        slot_removed: 'Se eliminó la ranura',
        slot_changed: 'Se reconfiguró la ranura',
        taken_over: 'Un administrador tomó esta ranura'
      },
      usb: {
        surrendered: 'El passthrough USB retiene el teclado y el ratón',
        surrenderedDesc:
          'El host remoto ve el dispositivo importado en lugar del teclado, el ratón y los medios virtuales del NanoKVM. Vuelven cuando la sesión termina.',
        unsupported: 'WebUSB necesita un navegador Chromium sobre HTTPS',
        session: 'Reenviando {{device}} ({{mode}})',
        idle: 'Ninguna sesión de passthrough',
        mode: {
          hybrid: 'híbrido',
          exact: 'exacto'
        }
      }
    },
    settings: {
      title: 'Ajustes',
      display: {
        title: 'Pantalla',
        loading: 'Cargando...',
        active: 'EDID activo',
        activeUnknown:
          'NanoKVM no ha escrito ningún EDID desde que arrancó, por lo que se desconoce la identidad que ve el host.',
        appliedAt: 'Aplicado el {{time}}',
        download: 'Descargar',
        downloadBackup: 'Descargar el anterior',
        preset: 'Preajuste de monitor',
        presetPlaceholder: 'Selecciona un monitor',
        upload: 'Subir',
        selected: 'EDID seleccionado',
        errors: 'Errores',
        warnings: 'Advertencias',
        info: 'Información',
        unknownMonitor: 'Monitor desconocido',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Audio',
        audioNo: 'Sin audio',
        extensionBlocks: 'Bloques de extensión: {{blocks}}',
        apply: 'Aplicar',
        applyTitle: '¿Aplicar este EDID?',
        before: 'Actual',
        after: 'Nuevo',
        hdmiNotice:
          'La captura de vídeo se detiene mientras se escribe el EDID y se reanuda sola al terminar.',
        powerCycleNotice:
          'Este dispositivo debe desconectarse físicamente de la corriente y volver a conectarse para que el nuevo EDID surta efecto.',
        powerCycleUnverified:
          'La escritura no se verificó, así que el chip de vídeo conserva lo que tiene ahora hasta que este dispositivo se desconecte físicamente de la corriente y se vuelva a conectar.',
        applied: 'EDID aplicado y verificado.',
        applyFailed: 'No se pudo aplicar el EDID.',
        busy: 'El chip de vídeo estaba ocupado. Inténtalo de nuevo.',
        unsupported: 'Este dispositivo no permite cambiar el EDID.',
        toolMissing: 'Falta la herramienta de EDID en este firmware.',
        noAudio: 'Este EDID no anuncia audio, por lo que el host puede dejar de enviar sonido.',
        oldVersion: 'Este EDID usa una versión anterior a la 1.4.',
        interlaced: 'La resolución preferida es entrelazada.',
        tooLarge:
          'La resolución preferida supera 1920x1080 a 60 Hz, más de lo que NanoKVM puede capturar.',
        recovery: 'Recuperación',
        recoveryNeeded:
          'La última escritura no se verificó, por lo que la zona EDID del chip de vídeo está en un estado desconocido. Restaura el EDID de fábrica para volver a un estado conocido.',
        recoveryDesc:
          'Vuelve a escribir un EDID conocido en el chip de vídeo cuando el que aplicaste dejó al host sin imagen.',
        restoreFactory: 'Restaurar EDID de fábrica',
        restoreBackup: 'Restaurar EDID anterior',
        restoreTitle: '¿Restaurar este EDID?',
        restoreFactoryTarget: 'El EDID de fábrica con el que se entrega NanoKVM.',
        restoreBackupTarget: 'La copia más reciente, aplicada el {{time}}.',
        restoreNotice:
          'Una restauración se escribe igual que una aplicación y tiene las mismas consecuencias.',
        restored: 'EDID restaurado y verificado.',
        restoreFailed: 'No se pudo restaurar el EDID.',
        mismatchTitle: 'Escrito y releído',
        mismatchWritten: 'Escrito',
        mismatchRead: 'Releído',
        restoreOkBtn: 'Restaurar',
        hardware: 'Hardware detectado: {{hardware}}',
        hardwareUnknown: 'Desconocido',
        confirmWord: 'APLICAR',
        confirmPrompt: 'Escribe {{word}} para habilitar el botón de aplicar.',
        okBtn: 'Aplicar',
        cancelBtn: 'Cancelar'
      },
      presentation: {
        title: 'Presentación USB',
        loading: 'Cargando...',
        current: 'Presentación USB actual',
        noProfile: 'Ningún perfil aplicado',
        linked: 'Funciones enlazadas',
        hostState: 'USB del host',
        hostUnbound: 'Controlador sin vincular',
        hdmiState: 'Entrada HDMI',
        hdmiSignal: 'Hay señal',
        hdmiUnreported: 'Todavía no hay informe de captura',
        endpoints: 'Endpoints',
        fifos: 'Ranuras FIFO',
        pending: 'Cambios pendientes',
        pendingEdits: 'Cambios de identidad sin guardar',
        pendingProfile: '{{profile}} está seleccionado pero no aplicado',
        pendingNone: 'Ninguno',
        lastApply: 'Última aplicación',
        applyFailed: 'Falló en {{profile}} el {{time}}',
        applyClean: 'Ningún fallo registrado',
        lastKnownGood: 'Último estado correcto conocido',
        rollbackTarget: 'Destino de la reversión',
        rollbackNone: 'Ninguno',
        powerCyclePending:
          'Se le quitó el controlador al host. Apaga y vuelve a encender el ordenador conectado para recuperar el dispositivo.',
        rollback: 'Revertir',
        rollbackTitle: '¿Revertir a {{profile}}?',
        rollbackDesc: 'El gadget se vuelve a enumerar; las funciones USB se caen brevemente.',
        profile: 'Perfil USB',
        builtIn: 'integrado',
        descriptors: 'descriptores',
        imported: 'importado',
        clone: 'Clonar',
        cloneTitle: 'Clonar este perfil',
        cloneToEdit:
          'Los perfiles integrados son de solo lectura. Clona este perfil para editar su identidad.',
        profileName: 'Nombre del perfil',
        profileNameHint: 'Letras minúsculas, números, puntos, guiones bajos y guiones.',
        import: 'Importar paquete',
        export: 'Exportar paquete',
        delete: 'Eliminar',
        deleteTitle: '¿Eliminar este perfil?',
        deleteDesc: 'Eliminar {{profile}} del NanoKVM.',
        identity: 'Identidad USB',
        preset: 'Identidad predefinida',
        presetPlaceholder: 'Copiar la identidad de un dispositivo conocido',
        presetHint:
          'Un preajuste rellena el Vendor ID, el Product ID y los dos campos de nombre. No aporta descriptores.',
        presetSource: 'Identidad tal como consta en {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'Este Vendor ID pertenece a otro fabricante',
        productId: 'Product ID',
        bcdUSB: 'Versión de USB',
        bcdDevice: 'Versión del dispositivo',
        manufacturer: 'Fabricante',
        product: 'Producto',
        serial: 'Número de serie',
        configuration: 'Cadena de configuración',
        hidLayout: 'Dispositivos HID',
        hidRoleKeyboard: 'Teclado',
        hidRoleRelative: 'Ratón (relativo)',
        hidRoleAbsolute: 'Puntero (absoluto)',
        hidOff: 'No presente',
        hidInterface: 'Interfaz {{index}}',
        hidBootKeyboardShared:
          'El teclado comparte una interfaz, así que ya no ofrece informe en protocolo boot. Algunas BIOS y UEFI no lo verán.',
        functions: 'Funciones',
        descriptorAssets: 'Descriptores almacenados: {{count}}',
        endpointUse:
          'IN {{inUse}} en uso, {{inFree}} libres; OUT {{outUse}} en uso, {{outFree}} libres',
        preview: 'Validar',
        save: 'Guardar',
        apply: 'Aplicar',
        applyTitle: '¿Aplicar este perfil USB?',
        applyDesc: 'El NanoKVM presentará {{profile}} al ordenador conectado.',
        reconnect:
          'El teclado, el ratón y las demás funciones USB se desconectan un instante mientras se vuelve a enlazar el gadget.',
        applyLinks: 'Enlaza: {{functions}}',
        applyRemoves: 'Elimina: {{functions}}',
        applyNoHid:
          'Tras esta aplicación no queda ninguna función HID. El teclado y el ratón dejarán de funcionar.',
        applyRollback: 'Si la aplicación falla, se vuelve a {{profile}}.',
        recoveryPowerCycle:
          'Ningún HID sobrevive a esta aplicación, así que un host que deje de responder solo se puede recuperar apagándolo y encendiéndolo.',
        recoveryReboot:
          'Una interfaz desaparece del dispositivo compuesto; puede que el host necesite reiniciarse para volver a vincular el resto.',
        recoveryHdmiReset:
          'Se reconstruye una función de vídeo, así que la cadena de captura que hay detrás se reinicia.',
        recoveryReconnect:
          'El host vuelve a enumerar el dispositivo; las funciones USB se caen brevemente.',
        cancel: 'Cancelar',
        noFunctions: 'Ninguna función enlazada',
        loadFailed: 'No se pudieron cargar los perfiles de presentación',
        operationFailed: 'La operación de presentación ha fallado'
      },
      passthrough: {
        title: 'Passthrough USB',
        loading: 'Cargando...',
        mode: 'Modo',
        hybrid: 'Híbrido',
        exact: 'Exacto',
        hybridDesc: 'Conserva el teclado boot y el ratón relativo, para dispositivos compatibles.',
        exactDesc: 'Sustituye todas las funciones USB de NanoKVM por el dispositivo importado.',
        hybridWarning: 'El modo híbrido mantiene el teclado y el ratón relativo',
        hybridWarningDesc:
          'El almacenamiento, la red USB y el puntero absoluto se desconectan mientras la función importada está activa.',
        hidWarning: 'Iniciar el passthrough cede el teclado, el ratón y los medios virtuales',
        hidWarningDesc:
          'NanoKVM tiene un único controlador de dispositivo USB y el proxy lo necesita entero, así que mientras haya una sesión el equipo remoto verá el dispositivo redirigido en lugar del teclado, el ratón y los medios virtuales de NanoKVM. Vuelven solos en cuanto se detiene la sesión. Esta interfaz web no se ve afectada, por lo que siempre puede detener la sesión desde esta página.',
        hidWarningSafeDesc:
          'NanoKVM tiene un único controlador de dispositivo USB y el proxy lo necesita entero, así que mientras haya una sesión el equipo remoto verá el dispositivo redirigido en lugar del teclado, el ratón y los medios virtuales de NanoKVM. Vuelven cuando se detiene la sesión.',
        isoLabel: 'Permitir transferencias isócronas',
        isoHint:
          'Deja pasar cámaras web, micrófonos y otros dispositivos de flujo. Nadie ha medido qué aguanta este hardware.',
        isoWarning:
          'El flujo isócrono no está probado aquí y puede retener el teclado y el ratón hasta que detengas la sesión',
        info: {
          title: 'Información',
          hybrid:
            'El modo híbrido mantiene disponibles el teclado y el ratón relativo. El almacenamiento, la red USB y el puntero absoluto se desconectan mientras el dispositivo importado está activo.',
          exact:
            'El modo exacto sustituye todas las funciones USB de NanoKVM por el dispositivo importado. El teclado, el ratón y los medios virtuales vuelven solos cuando se detiene la sesión.',
          udc: 'NanoKVM tiene un único controlador de dispositivo USB y el proxy lo necesita entero: por eso las funciones de arriba desaparecen mientras dura una sesión.',
          web: 'Esta interfaz web no se ve afectada, por lo que siempre puede detener la sesión desde esta página.',
          network:
            'Inicie el passthrough por Ethernet o Wi-Fi. Iniciarlo desde la red USB de NanoKVM se rechaza, porque esa conexión desaparecería.',
          iso: 'Las cámaras web, los micrófonos y otros dispositivos isócronos se rechazan mientras no permitas las transferencias isócronas. Esa vía funciona, pero nunca se ha medido en este hardware: considera su rendimiento desconocido.',
          camera:
            'La cámara y el micrófono del navegador, en Dispositivos, siguen siendo la forma probada de darle uno al equipo remoto.'
        },
        session: 'Sesión',
        activeDesc: 'Hay un dispositivo importado y el proxy mantiene el controlador USB.',
        inactiveDesc:
          'No hay ninguna sesión en curso. El teclado, el ratón y los medios virtuales funcionan con normalidad.',
        device: 'Dispositivo',
        busId: 'ID de bus',
        speed: 'Velocidad',
        exporter: 'Exportador',
        local: 'Importado como',
        localValue: 'Bus {{bus}}, dirección {{address}}',
        udc: 'Controlador USB',
        pid: 'PID del proxy',
        startedAt: 'Iniciada',
        isoDevice:
          'Este dispositivo transmite por puntos finales isócronos, algo que nunca se ha medido en este hardware',
        exporterLabel: 'Dirección del exportador',
        exporterHint:
          'El host y el puerto a los que llama NanoKVM. Con el túnel de abajo es {{exporter}}.',
        busIdLabel: 'ID de bus en su equipo',
        busIdHint:
          'El busid que usbip list -l muestra para el dispositivo, por ejemplo {{example}}.',
        start: 'Iniciar passthrough',
        stop: 'Detener passthrough',
        startTitle: '¿Iniciar el passthrough USB?',
        startDevice: 'NanoKVM importará {{busId}} desde {{exporter}}.',
        startHid:
          'El teclado USB, el ratón y los medios virtuales dejan de funcionar mientras dure la sesión y vuelven solos cuando la detenga.',
        startIso:
          'Las cámaras web y otros dispositivos isócronos necesitan que actives el interruptor isócrono antes de empezar.',
        startWeb:
          'Esta interfaz web sigue funcionando, así que puede detener la sesión desde esta página en cualquier momento.',
        startNetwork:
          'Use esta página por Ethernet o Wi-Fi. Iniciarla desde la red USB de NanoKVM se rechaza porque esa conexión desaparecería.',
        okBtn: 'Iniciar',
        cancelBtn: 'Cancelar',
        instructions: 'En su propio equipo',
        instructionsDesc:
          'Por diseño no hay ningún agente cliente que instalar. Ejecute estas órdenes estándar de usbip en el equipo al que está conectado el dispositivo.',
        copyFailed: 'No se pudo copiar. Copie la orden manualmente.',
        directNote:
          'Sin túnel, usbipd tiene que ser accesible en su red y la dirección del exportador de arriba debe apuntar a él. usbip transporta el dispositivo sin cifrar, así que es preferible el túnel.',
        steps: {
          modprobe: {
            title: 'Cargar el controlador del exportador',
            desc: 'usbip-host es lo que permite a su núcleo ceder un dispositivo local. No se carga por defecto.'
          },
          list: {
            title: 'Localizar el dispositivo',
            desc: 'Muestra todos los dispositivos locales con su busid y su par fabricante:producto. Anote el busid del que quiera.'
          },
          bind: {
            title: 'Enlazarlo a usbip',
            desc: 'Le quita el dispositivo a su controlador habitual, de modo que deja de funcionar en este equipo hasta que lo desenlace.'
          },
          serve: {
            title: 'Publicarlo',
            desc: 'usbipd se queda en primer plano y espera a que NanoKVM importe el dispositivo.',
            notice:
              'El usbipd estándar no tiene opción de dirección de escucha y escucha en todas las interfaces. Mantenga el puerto {{port}} cerrado en su cortafuegos y deje que solo lo alcance el túnel de abajo.'
          },
          tunnel: {
            title: 'Apuntarlo a NanoKVM',
            desc: 'Un túnel SSH inverso: el puerto {{port}} del bucle local de NanoKVM pasa a ser el usbipd de este equipo. Déjelo en marcha durante toda la sesión.'
          },
          exporter: {
            title: 'Usar esto como exportador',
            desc: 'Ponga esto en el campo del exportador de arriba, escriba el ID de bus e inicie la sesión.'
          },
          unbind: {
            title: 'Devolver el dispositivo',
            desc: 'Cuando la sesión se detenga, esto devuelve el dispositivo a su controlador habitual en este equipo.'
          }
        }
      },
      mcp: {
        title: 'Servicio MCP',
        service: 'Control remoto MCP',
        serviceDesc:
          'Permitir que clientes MCP de confianza controlen el teclado y el ratón y realicen capturas de pantalla',
        securityWarning:
          'Cualquier persona con esta clave API puede controlar el host remoto y ver su pantalla. Utiliza HTTPS y actívalo solo en redes de confianza.',
        endpoint: 'Punto de conexión',
        apiKey: 'Clave API',
        regenerateConfirmTitle: '¿Volver a generar la clave API de MCP?',
        regenerateConfirmDesc: 'La clave actual dejará de funcionar inmediatamente.',
        enableConfirmTitle: '¿Activar el control MCP externo?',
        enableConfirmDesc:
          'Al activar MCP se detendrá PicoClaw y se cerrará cualquier sesión activa de PicoClaw.',
        failed: 'Error en la operación MCP',
        copyFailed: 'Error al copiar. Copia manualmente.',
        okBtn: 'Confirmar',
        cancelBtn: 'Cancelar'
      },
      about: {
        title: 'Sobre NanoKVM',
        information: 'Información',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Versión de la aplicación',
        applicationTip: 'Versión de la aplicación web NanoKVM',
        image: 'Versión de la imagen',
        imageTip: 'Versión de la imagen del sistema NanoKVM',
        deviceKey: 'Clave del dispositivo',
        community: 'Comunidad',
        hostname: 'Nombre del host',
        hostnameUpdated: 'Nombre del host actualizado. Reinicia para aplicar.',
        ipType: {
          Wired: 'Cableada',
          Wireless: 'Inalámbrica',
          Other: 'Otra'
        }
      },
      appearance: {
        title: 'Apariencia',
        display: 'Pantalla',
        language: 'Idioma',
        languageDesc: 'Seleccionar el idioma de la interfaz',
        webTitle: 'Título web',
        webTitleDesc: 'Personaliza el título de la página web',
        menuBar: {
          title: 'Barra de menú',
          mode: 'Modo de visualización',
          modeDesc: 'Mostrar barra de menú en la pantalla',
          modeOff: 'Apagado',
          modeAuto: 'Ocultar automáticamente',
          modeAlways: 'Siempre visible',
          keyboardLedStatus: 'Indicadores de bloqueo del teclado',
          keyboardLedStatusDesc:
            'Mostrar el estado de Bloq Num, Bloq Mayús y Bloq Despl del equipo remoto',
          icons: 'Iconos del submenú',
          iconsDesc: 'Mostrar iconos de submenú en la barra de menú'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Estado de bloqueos del teclado remoto',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Bloq Num',
        numLockShort: 'Num',
        capsLock: 'Bloq Mayús',
        capsLockShort: 'May',
        scrollLock: 'Bloq Despl',
        scrollLockShort: 'Despl',
        on: 'Activado',
        off: 'Desactivado',
        unknown: 'Desconocido'
      },
      device: {
        title: 'Dispositivo',
        oled: {
          title: 'OLED',
          description: 'La pantalla OLED entra en reposo automáticamente',
          0: 'Nunca',
          15: '15 s',
          30: '30 s',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 hora'
        },
        ssh: {
          description: 'Habilitar acceso remoto SSH',
          tip: 'Establece una contraseña segura antes de habilitar (Cuenta - Cambiar contraseña)'
        },
        advanced: 'Ajustes avanzados',
        swap: {
          title: 'Memoria Swap',
          disable: 'Desactivar',
          description: 'Establece el tamaño del archivo swap',
          tip: 'Habilitar esta función podría acortar la vida útil de tu tarjeta SD.'
        },
        mouseJiggler: {
          title: 'Mouse Jiggler',
          description: 'Evitar que el host remoto entre en reposo',
          disable: 'Desactivar',
          absolute: 'Modo absoluto',
          relative: 'Modo relativo'
        },
        mdns: {
          description: 'Habilitar servicio de descubrimiento mDNS',
          tip: 'Desactívalo si no es necesario'
        },
        hdmi: {
          description: 'Habilitar salida HDMI/monitor',
          idleTimeoutTitle: 'Tiempo de espera de captura inactiva',
          idleTimeoutDescription:
            'Detener la captura HDMI después de no haber espectadores activos durante',
          minutes: 'min'
        },
        autostart: {
          title: 'Configuración de scripts de inicio automático',
          description: 'Administrar scripts que se ejecutan automáticamente al iniciar el sistema',
          new: 'Nuevo',
          deleteConfirm: '¿Estás seguro de que deseas eliminar este archivo?',
          yes: 'Sí',
          no: 'No',
          scriptName: 'Nombre del script de inicio automático',
          scriptContent: 'Contenido del script de inicio automático',
          settings: 'Ajustes'
        },
        hidOnly: 'Modo sólo HID',
        hidOnlyDesc:
          'Dejar de emular dispositivos virtuales y conservar solo el control básico HID',
        disk: 'Disco Virtual',
        diskDesc: 'Montar disco virtual en el host remoto',
        network: 'Red Virtual',
        networkDesc: 'Montar tarjeta de red virtual en el host remoto',
        networkProtocol: 'Protocolo de red',
        networkProtocolDesc: 'NCM para hosts modernos, RNDIS para Windows antiguos',
        media: {
          title: 'Ranuras de cámara y micrófono',
          desc: 'Declare los dispositivos multimedia que los navegadores pueden ocupar. El presupuesto de endpoints se comprueba al aplicar el perfil USB. Al guardar, el dispositivo se vuelve a enumerar y se desconecta cualquier navegador conectado.',
          cameras: 'Cámaras',
          microphones: 'Micrófonos',
          name: 'Nombre',
          namePlaceholder: 'Se muestra en el equipo de destino',
          addCamera: 'Añadir cámara',
          addMicrophone: 'Añadir micrófono',
          remove: 'Quitar',
          cameraDefault: 'Cámara NanoKVM {{index}}',
          microphoneDefault: 'Micrófono NanoKVM {{index}}',
          nameRequired: 'Cada ranura necesita un nombre.',
          unsupported:
            'Este kernel no puede nombrar los dispositivos multimedia, así que los equipos muestran el nombre predeterminado.',
          save: 'Guardar ranuras',
          disconnect: 'Desconectar',
          disconnectAll: 'Desconectar todas las fuentes',
          limit: 'Las ranuras de cámara y micrófono no pueden sumar más de ocho.',
          failed: 'No se pudieron actualizar las ranuras multimedia.'
        },
        reboot: 'Reiniciar',
        rebootDesc: '¿Estás seguro de que deseas reiniciar el NanoKVM?',
        okBtn: 'Sí',
        cancelBtn: 'No'
      },
      network: {
        title: 'Red',
        wifi: {
          title: 'Wi-Fi',
          description: 'Configura el Wi-Fi',
          apMode: 'El modo AP está activado; conéctate al Wi-Fi escaneando el código QR',
          connect: 'Conectar Wi-Fi',
          connectDesc1: 'Introduce el SSID de la red y la contraseña',
          connectDesc2: 'Introduce la contraseña para unirte a esta red',
          disconnect: '¿Seguro que quieres desconectar la red?',
          failed: 'Error de conexión, inténtalo de nuevo.',
          ssid: 'Nombre',
          password: 'Contraseña',
          joinBtn: 'Unirse',
          confirmBtn: 'Aceptar',
          cancelBtn: 'Cancelar'
        },
        tls: {
          description: 'Habilitar protocolo HTTPS',
          tip: 'Aviso: Usar HTTPS puede aumentar la latencia, especialmente en modo de vídeo MJPEG.'
        },
        bridge: {
          title: 'Puente de red',
          twoDevices:
            'Tu router verá NanoKVM y el ordenador controlado como dos dispositivos independientes, cada uno con su propia dirección.',
          loading: 'Cargando...',
          state: 'Estado',
          states: {
            disabled: 'Desactivado',
            enabled: 'Activado',
            rolledBack: 'Revertido',
            failed: 'Fallido',
            pending: 'En curso'
          },
          uplink: 'Enlace ascendente',
          ports: 'Puertos',
          protocol: 'Protocolo del dispositivo',
          up: 'activo',
          down: 'inactivo',
          noLink: 'sin enlace',
          enableTitle: '¿Activar el puente de red?',
          disableTitle: '¿Desactivar el puente de red?',
          reconnect:
            'La conexión de administración se desconectará y volverá a conectarse brevemente mientras se mueve la dirección.',
          rollback:
            'Si la verificación falla, se restaura automáticamente la configuración anterior.',
          enableBtn: 'Activar',
          disableBtn: 'Desactivar',
          cancelBtn: 'Cancelar',
          interrupted:
            'La conexión se interrumpió durante la aplicación. Comprobando de nuevo el estado actual.',
          pendingNotice: 'Un cambio del puente sigue en curso o se interrumpió antes de terminar.',
          revert: 'Restaurar la configuración anterior',
          rolledBackNotice: 'El último cambio se revirtió y se restauró la configuración anterior.',
          verifyFailed: 'Error de verificación: {{gates}}',
          gates: {
            address: 'dirección',
            gateway: 'puerta de enlace',
            inbound: 'conexión entrante'
          },
          inboundWeak:
            'La comprobación entrante solo pasó porque NanoKVM se conectó a sí mismo. Eso demuestra que el servicio web está escuchando y es accesible localmente, no que llegue una petición desde la red.',
          noCarrier:
            'No hay enlace en {{port}}. El puente no tiene ningún camino hacia la red hasta que se conecte un cable.',
          loop: 'El router también se está aprendiendo en {{port}}, así que ese puerto es un segundo camino hacia la misma red. El spanning tree está desactivado, de modo que nada de aquí romperá el bucle: desconecta uno de los dos caminos.',
          failedNotice:
            'No se pudo deshacer el último cambio. Puede que solo se pueda acceder a NanoKVM por el AP Wi-Fi o una consola serie.'
        },
        dns: {
          title: 'DNS',
          description: 'Configura los servidores DNS para NanoKVM',
          mode: 'Modo',
          dhcp: 'DHCP',
          manual: 'Manual',
          add: 'Añadir DNS',
          save: 'Guardar',
          invalid: 'Introduce una dirección IP válida',
          noDhcp: 'No hay DNS DHCP disponible actualmente',
          saved: 'Configuración DNS guardada',
          saveFailed: 'No se pudo guardar la configuración DNS',
          unsaved: 'Cambios sin guardar',
          maxServers: 'Se permiten como máximo {{count}} servidores DNS',
          dnsServers: 'Servidores DNS',
          dhcpServersDescription: 'Los servidores DNS se obtienen automáticamente por DHCP',
          manualServersDescription: 'Los servidores DNS se pueden editar manualmente',
          networkDetails: 'Detalles de red',
          interface: 'Interfaz',
          ipAddress: 'Dirección IP',
          subnetMask: 'Máscara de subred',
          router: 'Router',
          none: 'Ninguno'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'Servidor VNC',
        description:
          'Permite que cualquier cliente VNC vea la pantalla remota y use el teclado y el ratón, iniciando sesión con su cuenta de NanoKVM',
        port: 'Puerto',
        portDescription: 'Conéctese a este puerto en la dirección del NanoKVM'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Optimización de memoria',
          tip: 'Cuando el uso de memoria supera el límite, la recolección de basura se ejecuta de forma más agresiva para intentar liberar memoria. Es necesario reiniciar Tailscale para que el cambio surta efecto.'
        },
        swap: {
          title: 'Memoria Swap',
          tip: 'Si los problemas persisten después de habilitar la optimización de memoria, prueba a activar la memoria swap. Esto establece el tamaño del archivo swap en 256 MB por defecto, que se puede ajustar en "Ajustes > Dispositivo".'
        },
        restart: '¿Seguro que deseas reiniciar Tailscale?',
        stop: '¿Seguro que deseas detener Tailscale?',
        stopDesc: 'Cerrar sesión en Tailscale y desactivar su inicio automático al arrancar.',
        loading: 'Cargando...',
        notInstall: '¡Tailscale no encontrado! Por favor, instálalo.',
        install: 'Instalar',
        installing: 'Instalando',
        failed: 'La instalación falló',
        retry:
          'Por favor, actualiza la página e inténtalo de nuevo. O intenta instalarlo manualmente',
        download: 'Descargar el',
        package: 'paquete de instalación',
        unzip: 'y descomprimirlo',
        upTailscale: 'Sube tailscale al directorio /usr/bin/ del NanoKVM',
        upTailscaled: 'Sube tailscaled al directorio /usr/sbin/ del NanoKVM',
        refresh: 'Actualizar la página actual',
        notRunning: 'Tailscale no se está ejecutando. Por favor, inícialo para continuar.',
        run: 'Iniciar',
        notLogin:
          'El dispositivo aún no ha sido vinculado. Por favor, inicia sesión y vincula este dispositivo a tu cuenta.',
        urlPeriod: 'Esta URL es válida por 10 minutos',
        login: 'Iniciar sesión',
        loginSuccess: 'Inicio de sesión exitoso',
        enable: 'Habilitar Tailscale',
        deviceName: 'Nombre del dispositivo',
        deviceIP: 'IP del dispositivo',
        account: 'Cuenta',
        logout: 'Cerrar sesión',
        logoutDesc: '¿Estás seguro de que deseas cerrar sesión?',
        uninstall: 'Desinstalar Tailscale',
        uninstallDesc: '¿Estás seguro de que deseas desinstalar Tailscale?',
        okBtn: 'Sí',
        cancelBtn: 'No'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: 'Cargando...',
        notInstall: 'No instalado',
        notConfigured: 'Sin configurar',
        stopped: 'Detenido',
        running: 'En ejecución',
        connected: 'Conectado',
        error: 'Error',
        atBoot: 'se inicia al arrancar',
        notAtBoot: 'no se inicia al arrancar',
        arguments: 'Argumentos',
        argumentsTip: 'Argumentos de línea de comandos que se pasan al servicio al iniciarse.',
        env: 'Variables de entorno',
        envKey: 'Nombre',
        envValue: 'Valor',
        envAdd: 'Añadir variable',
        envRemove: 'Eliminar',
        configured: 'Configurada',
        save: 'Guardar',
        saved: 'Configuración guardada',
        start: 'Iniciar',
        stop: 'Detener',
        restart: 'Reiniciar',
        logs: 'Registros',
        logsEmpty: 'Todavía no hay registros',
        refresh: 'Actualizar',
        binary: 'Binario',
        binaryShipped: 'Incluido en el firmware',
        binaryCustom: 'Binario personalizado',
        binaryUpload: 'Subir binario',
        binaryRevert: 'Restaurar el binario incluido',
        binaryRevertDesc: '¿Eliminar el binario subido y restaurar el incluido en el firmware?',
        serverWarning: 'Un servidor sin restricciones actúa como proxy abierto',
        noHealthSignal:
          'Este servicio no informa de su estado, así que NanoKVM solo sabe que el proceso está en marcha, no si el túnel está conectado.',
        memoryWarning:
          'Ejecutar varios servicios de acceso remoto a la vez puede agotar la memoria',
        resources: 'Recursos',
        memory: {
          title: 'Límite de memoria',
          description:
            'Limita el heap de Go de newt a {{limit}} MiB a partir de su próximo reinicio. Su propio límite, no el de Tailscale; desactivado se usa el valor por defecto de Go, con GOGC=50 en ambos casos.',
          noRuntime:
            'wstunnel está escrito en Rust: no hay recolector de basura ni límite de heap que fijar, y sus hilos de trabajo ya se ajustan a la única CPU del dispositivo.',
          notApplicable: 'No aplicable'
        },
        swap: {
          title: 'Archivo de intercambio',
          description:
            'Añade un archivo de intercambio de 256 MB en la tarjeta SD. Es de todo el sistema: el mismo intercambio sirve a Tailscale, al servidor KVM y a todo lo demás del dispositivo.'
        },
        okBtn: 'Sí',
        cancelBtn: 'No'
      },
      update: {
        title: 'Buscar actualizaciones',
        queryFailed: 'Error al obtener la versión',
        updateFailed: 'La actualización falló. Por favor, inténtalo de nuevo.',
        isLatest: 'Ya tienes la última versión.',
        rebooting:
          'Instalando el nuevo kernel y reiniciando. Puede tardar unos minutos; no apagues el dispositivo.',
        kernelUpdate:
          'Esta actualización instala el kernel {{version}}. El dispositivo se reiniciará y volverá solo al kernel actual si el nuevo no arranca.',
        rolledBack: 'El kernel {{version}} no arrancó y el dispositivo volvió al kernel anterior.',
        available: 'Hay una actualización disponible. ¿Estás seguro de que quieres actualizar?',
        updating: 'Actualización iniciada. Por favor, espera...',
        confirm: 'Confirmar',
        cancel: 'Cancelar',
        preview: 'Vista previa de actualizaciones',
        previewDesc: 'Accede anticipadamente a nuevas funciones y mejoras',
        previewTip:
          'Ten en cuenta que las versiones de vista previa pueden contener errores o funcionalidades incompletas',
        customServer: {
          title: 'Servidor de actualizaciones personalizado',
          desc: 'Buscar y descargar actualizaciones en línea desde un servidor especificado',
          invalidUrl:
            'Introduce un directorio de servidor HTTP o HTTPS válido, sin parámetros de consulta, fragmentos ni latest.json.',
          loadFailed: 'No se pudo cargar la configuración del servidor de actualizaciones.',
          saveFailed: 'No se pudo guardar la configuración del servidor de actualizaciones.',
          saved: 'Se ha guardado la configuración del servidor de actualizaciones.',
          save: 'Guardar',
          confirmTitle: '¿Usar un servidor de actualizaciones personalizado?',
          confirmDesc:
            'SHA-512 solo comprueba que el paquete coincide con el manifiesto proporcionado por este servidor. No demuestra que el paquete sea una versión oficial de NanoKVM. Un servidor defectuoso o malicioso puede inutilizar el dispositivo, provocar la pérdida de datos o comprometer el sistema.',
          confirm: 'Usar de todos modos',
          previewDisabled:
            'Las actualizaciones preliminares no están disponibles mientras esté activado un servidor de actualizaciones personalizado.'
        },
        offline: {
          title: 'Actualizaciones sin conexión',
          desc: 'Actualización a través del paquete de instalación local',
          upload: 'Subir',
          checksumPlaceholder: 'Suma de comprobación SHA-256 (opcional)',
          invalidChecksum:
            'La suma de comprobación SHA-256 debe contener 64 caracteres hexadecimales.',
          checksumMismatch:
            'La verificación SHA-256 ha fallado. Es posible que el paquete esté dañado.',
          invalidName:
            'Formato de nombre de archivo no válido. Descargue desde las versiones de GitHub.',
          updateFailed: 'La actualización falló. Por favor, inténtalo de nuevo.'
        }
      },
      account: {
        title: 'Cuenta',
        webAccount: 'Nombre de la cuenta web',
        role: 'Rol',
        roles: {
          admin: 'Administrador',
          user: 'Usuario'
        },
        password: 'Contraseña',
        updateBtn: 'Actualizar',
        logoutBtn: 'Cerrar sesión',
        logoutDesc: '¿Estás seguro de que deseas cerrar sesión?',
        okBtn: 'Sí',
        cancelBtn: 'No',
        users: {
          title: 'Usuarios',
          create: 'Crear usuario',
          enabled: 'Habilitado',
          disabled: 'Deshabilitado',
          deviceOwner: 'Propietario del dispositivo',
          resetPassword: 'Restablecer contraseña',
          delete: 'Eliminar',
          deleteConfirm: '¿Eliminar este usuario y revocar todas sus sesiones?',
          created: 'Usuario creado',
          deleted: 'Usuario eliminado',
          passwordUpdated: 'Contraseña actualizada',
          loadFailed: 'No se pudieron cargar los usuarios',
          saveFailed: 'No se pudo guardar el usuario',
          deleteFailed: 'No se pudo eliminar el usuario'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Asistente',
      empty: 'Abre el panel e inicia una tarea para comenzar.',
      inputPlaceholder: 'Describe lo que quieres que haga el PicoClaw',
      newConversation: 'Nueva conversación',
      processing: 'Procesando...',
      agent: {
        defaultTitle: 'Asistente general',
        defaultDescription: 'Ayuda general para chat, búsqueda y espacio de trabajo.',
        kvmTitle: 'Control remoto',
        kvmDescription: 'Opere el host remoto a través de NanoKVM.',
        switched: 'Rol de agente cambiado',
        switchFailed: 'No se pudo cambiar la función del agente'
      },
      send: 'Enviar',
      cancel: 'Cancelar',
      status: {
        connecting: 'Conectándose a la puerta de enlace...',
        connected: 'Sesión de PicoClaw conectada',
        disconnected: 'Sesión de PicoClaw cerrada',
        stopped: 'Solicitud de detención enviada',
        runtimeStarted: 'Tiempo de ejecución de PicoClaw iniciado',
        runtimeStartFailed: 'Error al iniciar el tiempo de ejecución de PicoClaw',
        runtimeStopped: 'Tiempo de ejecución de PicoClaw detenido',
        runtimeStopFailed: 'No se pudo detener el tiempo de ejecución de PicoClaw',
        controlSwitchedToMCP: 'El control se ha transferido al servicio MCP externo'
      },
      connection: {
        runtime: {
          checking: 'Comprobando',
          restoring: 'Restoring PicoClaw',
          ready: 'Tiempo de ejecución listo',
          stopped: 'Tiempo de ejecución detenido',
          blockedByMCP: 'El control MCP externo está activo',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Tiempo de ejecución no disponible',
          configError: 'Error de configuración'
        },
        transport: {
          connecting: 'Conectando',
          connected: 'Conectado',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Inactivo',
          busy: 'Ocupado'
        }
      },
      message: {
        toolAction: 'Acción',
        observation: 'Observación',
        screenshot: 'Captura de pantalla'
      },
      overlay: {
        locked: 'PicoClaw está controlando el dispositivo. La entrada manual está en pausa.'
      },
      control: {
        picoclaw: 'Control del dispositivo: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Control del dispositivo: MCP externo',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Control del dispositivo: desactivado',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Conceder control',
        release: 'Liberar',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'Control de PicoClaw concedido',
        released: 'Control de PicoClaw liberado',
        grantFailed: 'No se pudo conceder el control de PicoClaw',
        releaseFailed: 'No se pudo liberar el control de PicoClaw',
        grantConfirmTitle: '¿Cambiar el control del dispositivo a PicoClaw?',
        grantConfirmDesc: 'Se interrumpirán las escrituras de dispositivo del MCP externo.'
      },
      install: {
        install: 'Instalar PicoClaw',
        installing: 'Instalando PicoClaw',
        success: 'PicoClaw instalado correctamente',
        failed: 'Error al instalar PicoClaw',
        uninstalling: 'Desinstalando el tiempo de ejecución...',
        uninstalled: 'El tiempo de ejecución se desinstaló exitosamente.',
        uninstallFailed: 'Falló la desinstalación.',
        requiredTitle: 'PicoClaw no está instalado',
        requiredDescription:
          'Instala PicoClaw antes de iniciar el tiempo de ejecución de PicoClaw.',
        progressDescription: 'PicoClaw se está descargando e instalando.',
        stages: {
          preparing: 'Preparando',
          downloading: 'Descargando',
          extracting: 'Extrayendo',
          verifying: 'Verificando',
          installing: 'Instalando',
          installed: 'Instalado',
          install_timeout: 'Tiempo de espera agotado',
          install_failed: 'Falló'
        }
      },
      model: {
        requiredTitle: 'Se requiere configuración del modelo',
        requiredDescription: 'Configure el modelo PicoClaw antes de usar el chat PicoClaw.',
        docsTitle: 'Guía de configuración',
        docsDesc: 'Modelos y protocolos compatibles',
        menuLabel: 'Configurar modelo',
        modelIdentifier: 'Identificador de modelo',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'Clave API',
        apiKeyPlaceholder: 'Introduzca la clave API del modelo',
        save: 'Guardar',
        saving: 'Guardando',
        saved: 'Configuración del modelo guardada',
        saveFailed: 'No se pudo guardar la configuración del modelo',
        invalid: 'Se requieren el identificador del modelo, API Base URL y la clave API'
      },
      uninstall: {
        menuLabel: 'Desinstalar',
        confirmTitle: 'Desinstalar PicoClaw',
        confirmContent:
          '¿Está seguro de que desea desinstalar PicoClaw? Esto eliminará el ejecutable y todos los archivos de configuración.',
        confirmOk: 'Desinstalar',
        confirmCancel: 'Cancelar'
      },
      history: {
        title: 'Historial',
        loading: 'Cargando sesiones...',
        emptyTitle: 'Aún no hay historial',
        emptyDescription: 'Las sesiones PicoClaw anteriores aparecerán aquí.',
        loadFailed: 'Error al cargar el historial de sesiones',
        deleteFailed: 'No se pudo eliminar la sesión',
        deleteConfirmTitle: 'Eliminar sesión',
        deleteConfirmContent: '¿Está seguro de que desea eliminar "{{title}}"?',
        deleteConfirmOk: 'Eliminar',
        deleteConfirmCancel: 'Cancelar',
        messageCount_one: '{{count}} mensaje',
        messageCount_other: '{{count}} mensajes',
        messageCount: '{{count}} mensajes'
      },
      config: {
        startRuntime: 'Iniciar PicoClaw',
        stopRuntime: 'Detener PicoClaw'
      },
      start: {
        enableConfirmTitle: '¿Cambiar el control a PicoClaw?',
        enableConfirmDesc: 'Al iniciar PicoClaw se desactivará el servicio MCP externo.',
        enableConfirmOk: 'Iniciar PicoClaw',
        enableConfirmCancel: 'Cancelar',
        title: 'Iniciar PicoClaw',
        description:
          'Inicia el tiempo de ejecución para comenzar a utilizar el asistente PicoClaw.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Hemos encontrado un problema',
      refresh: 'Actualizar'
    },
    fullscreen: {
      toggle: 'Activar/Desactivar pantalla completa'
    },
    menu: {
      collapse: 'Colapsar menú',
      expand: 'Expandir menú'
    }
  }
};

export default es;
