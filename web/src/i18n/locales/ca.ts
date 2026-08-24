const ca = {
  translation: {
    head: {
      desktop: 'Escriptori remot',
      login: 'Inici de sessió',
      changePassword: 'Canviar contrasenya',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Inici de sessió',
      placeholderUsername: "Nom d'usuari",
      placeholderPassword: 'Contrasenya',
      placeholderCurrentPassword: 'Contrasenya actual',
      placeholderPassword2: 'Torna a introduir la contrasenya',
      noEmptyUsername: "Cal introduir el nom d'usuari",
      noEmptyPassword: 'Cal introduir la contrasenya',
      passwordLength: 'La contrasenya ha de tenir entre 8 i 72 caràcters',
      noAccount:
        "No s'ha pogut obtenir la informació de l'usuari, actualitza la pàgina web o restableix la contrasenya",
      invalidUser: "Nom d'usuari o contrasenya invàlids",
      locked: 'Massa inicis de sessió, si us plau, torna-ho a provar més tard',
      globalLocked: 'Sistema sota protecció, torneu-ho a provar més tard',
      error: 'Error inesperat',
      invalidCurrentPassword: 'La contrasenya actual no és correcta',
      changePassword: 'Canviar la contrasenya',
      changePasswordDesc: 'Per a la seguretat del dispositiu, canvia la contrasenya!',
      differentPassword: 'Les contrasenyes no coincideixen',
      illegalUsername: "El nom d'usuari conté caràcters no permesos",
      illegalPassword: 'La contrasenya conté caràcters no permesos',
      forgetPassword: 'Has oblidat la contrasenya',
      ok: "D'acord",
      cancel: 'Cancel·la',
      loginButtonText: 'Inicia sessió',
      tips: {
        reset1:
          'Per restablir les contrasenyes, mantingues premut el botó BOOT del NanoKVM durant 10 segons.',
        reset2: 'Per veure els passos detallats, consulta aquest document:',
        reset3: 'Compte web per defecte:',
        reset4: 'Compte SSH per defecte:',
        change1: 'Aquesta acció canviarà les següents contrasenyes:',
        change2: "Contrasenya d'inici de sessió web",
        change3: 'Contrasenya root del sistema (inici de sessió SSH)',
        change4: 'Per restablir les contrasenyes, mantingues premut el botó BOOT del NanoKVM.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Configura Wi-Fi per al NanoKVM',
      success: "Comprova l'estat de la xarxa del NanoKVM i visita la nova adreça IP.",
      failed: "L'operació ha fallat, torna-ho a intentar.",
      invalidMode:
        'El mode actual no admet la configuració de xarxa. Aneu al vostre dispositiu i activeu el mode de configuració Wi-Fi.',
      confirmBtn: "D'acord",
      finishBtn: 'Fet',
      ap: {
        authTitle: 'Es requereix autenticació',
        authDescription: 'Introduïu la contrasenya AP per continuar',
        authFailed: 'Contrasenya AP no vàlida',
        passPlaceholder: 'AP contrasenya',
        verifyBtn: 'Verificar'
      }
    },
    screen: {
      scale: 'Escala',
      title: 'Pantalla',
      video: 'Mode de vídeo',
      videoDirectTips: "Activa HTTPS a 'Configuració > Dispositiu' per utilitzar aquest mode",
      resolution: 'Resolució',
      auto: 'Automàtic',
      autoTips:
        "Poden aparèixer talls o desajustos del ratolí en certes resolucions. Prova a canviar la resolució de l'amfitrió remot o desactiva el mode automàtic.",
      fps: 'FPS',
      customizeFps: 'Personalitzat',
      quality: 'Qualitat',
      qualityLossless: 'Sense pèrdua',
      qualityHigh: 'Alta',
      qualityMedium: 'Mitjana',
      qualityLow: 'Baixa',
      frameDetect: 'Detecció de fotogrames',
      frameDetectTip:
        "Calcula la diferència entre fotogrames. S'atura la transmissió si no hi ha canvis a la pantalla de l'amfitrió remot.",
      resetHdmi: 'Restablir HDMI',
      mixedH264: {
        title: 'Conflicte de flux H.264',
        description:
          "S'estan utilitzant H.264 Direct i H.264 WebRTC alhora. Això pot provocar esquinçament de pantalla o vídeo corrupte. Utilitzeu només un mode H.264."
      },
      webrtcConnectionFailed: {
        title: 'Error de connexió WebRTC',
        description: 'Comproveu la connexió de xarxa o canvieu el mode de vídeo.'
      },
      captureStatus: {
        hdmiError: 'Error a la pantalla HDMI',
        unsupportedResolution: 'La resolució actual no és compatible',
        retrieving: "S'està obtenint la pantalla...",
        changingResolution: "S'està canviant la resolució...",
        updateFailed: 'La pantalla no es pot actualitzar ara',
        videoError: 'Error de visualització de vídeo',
        noHdmi: "No s'ha detectat cap senyal HDMI",
        unavailable: 'La pantalla no es pot mostrar ara'
      }
    },
    keyboard: {
      title: 'Teclat',
      paste: 'Enganxa',
      tips: 'Només es permeten lletres i símbols estàndard',
      placeholder: 'Escriu aquí',
      submit: 'Envia',
      virtual: 'Teclat',
      readClipboard: 'Llegir des del porta-retalls',
      clipboardPermissionDenied:
        "S'ha denegat el permís del porta-retalls. Permet l'accés al porta-retalls al teu navegador.",
      clipboardReadError: "No s'ha pogut llegir el porta-retalls",
      dropdownEnglish: 'Anglès',
      dropdownGerman: 'alemany',
      dropdownFrench: 'francès',
      dropdownRussian: 'rus',
      shortcut: {
        title: 'Dreceres',
        custom: 'Personalitzat',
        capture: 'Feu clic aquí per capturar la drecera',
        clear: 'Clar',
        save: 'Desa',
        captureTips:
          'Capturar tecles del sistema (com la tecla Windows) requereix permís de pantalla completa.',
        enterFullScreen: 'Canvia el mode de pantalla completa.'
      },
      leaderKey: {
        title: 'Tecla líder',
        desc: "Evita les restriccions del navegador i envia dreceres del sistema directament a l'amfitrió remot.",
        howToUse: "Com s'utilitza",
        simultaneous: {
          title: 'Mode simultània',
          desc1: 'Manteniu premuda la tecla líder i premeu la drecera.',
          desc2: 'Intuïtiu, però pot entrar en conflicte amb les dreceres del sistema.'
        },
        sequential: {
          title: 'Mode seqüencial',
          desc1:
            'Premeu la tecla líder → premeu la drecera en seqüència → torneu a prémer la tecla líder.',
          desc2: 'Requereix més passos, però evita completament els conflictes del sistema.'
        },
        enable: 'Activa la tecla líder',
        tip: "Quan s'assigna com a tecla líder, aquesta tecla només funciona com a activador de dreceres i perd el seu comportament predeterminat.",
        placeholder: 'Premeu la tecla líder',
        shiftRight: 'Maj dreta',
        ctrlRight: 'Ctrl dret',
        metaRight: 'Win dret',
        submit: 'Envia',
        recorder: {
          rec: 'REC',
          activate: 'Activa les tecles',
          input: 'Premeu la drecera...'
        }
      }
    },
    mouse: {
      title: 'Ratolí',
      cursor: 'Estil del cursor',
      default: 'Cursor per defecte',
      pointer: 'Cursor punter',
      cell: 'Cursor de cel·la',
      text: 'Cursor de text',
      grab: 'Cursor de mà',
      hide: 'Amaga el cursor',
      mode: 'Mode de ratolí',
      absolute: 'Mode absolut',
      relative: 'Mode relatiu',
      direction: 'Direcció de la roda de desplaçament',
      scrollUp: "Desplaça't cap amunt",
      scrollDown: "Desplaça't cap avall",
      speed: 'Velocitat de desplaçament',
      fast: 'Ràpida',
      slow: 'Lenta',
      requestPointer: "Estàs usant el mode relatiu. Fes clic a l'escriptori per obtenir el punter.",
      resetHid: 'Restablir HID',
      hidOnly: {
        title: 'Mode només HID',
        desc: 'Si el ratolí i el teclat deixen de respondre i restablir HID no ajuda, pot ser un problema de compatibilitat entre el NanoKVM i el dispositiu. Proveu d’activar el mode només HID per millorar la compatibilitat.',
        tip1: 'Activar el mode només HID desmuntarà el disc virtual i la xarxa virtual',
        tip2: 'En mode només HID, no es pot muntar imatges',
        tip3: 'El NanoKVM es reiniciarà automàticament en canviar de mode',
        enable: 'Activa mode només HID',
        disable: 'Desactiva mode només HID'
      }
    },
    image: {
      title: 'Imatges',
      loading: 'Carregant...',
      empty: "No s'ha trobat res",
      mountMode: 'Mode de muntatge',
      mountFailed: 'Error en muntar',
      mountDesc: 'En alguns sistemes cal expulsar el disc virtual abans de muntar la imatge.',
      unmountFailed: "No s'ha pogut desmuntar",
      unmountDesc:
        "En alguns sistemes, cal expulsar manualment de l'amfitrió remot abans de desmuntar la imatge.",
      refresh: 'Actualitza la llista',
      attention: 'Atenció',
      deleteConfirm: 'Esteu segur que voleu suprimir aquesta imatge?',
      okBtn: 'Sí',
      cancelBtn: 'No',
      tips: {
        title: 'Com pujar imatges',
        usb1: 'Connecta el NanoKVM al teu ordinador via USB.',
        usb2: "Assegura't que el disc virtual està muntat (Configuració - Disc Virtual).",
        usb3: "Obre el disc virtual i copia la imatge a l'arrel.",
        scp1: 'Comprova que el NanoKVM i el teu ordinador estan a la mateixa xarxa.',
        scp2: 'Obre un terminal i usa SCP per pujar la imatge al directori /data del NanoKVM.',
        scp3: 'Exemple: scp ruta-de-la-imatge root@ip-del-nanokvm:/data',
        tfCard: 'Targeta TF',
        tf1: 'Mètode disponible a sistemes GNU/Linux',
        tf2: 'Extreu la targeta TF del NanoKVM (versió FULL, cal obrir la carcassa).',
        tf3: "Introdueix la targeta en un lector i connecta'l a l'ordinador.",
        tf4: 'Copia la imatge al directori /data de la targeta.',
        tf5: 'Reintrodueix la targeta al NanoKVM.'
      }
    },
    script: {
      title: 'Scripts',
      upload: 'Puja',
      run: 'Executa',
      runBackground: 'Executa en segon pla',
      runFailed: "Error en l'execució",
      attention: 'Atenció',
      delDesc: 'Estàs segur que vols eliminar aquest fitxer?',
      confirm: 'Sí',
      cancel: 'No',
      delete: 'Esborra',
      close: 'Tanca'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'Terminal NanoKVM',
      serial: 'Terminal de port sèrie',
      serialPort: 'Port sèrie',
      serialPortPlaceholder: 'Introdueix el port sèrie',
      baudrate: 'Velocitat (baudrate)',
      parity: 'Paritat',
      parityNone: 'Cap',
      parityEven: 'Parell',
      parityOdd: 'Senar',
      flowControl: 'Control de flux',
      flowControlNone: 'Cap',
      flowControlSoft: 'Programari',
      flowControlHard: 'Maquinari',
      dataBits: 'Bits de dades',
      stopBits: 'Bits de parada',
      confirm: "D'acord"
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Enviant comanda...',
      sent: 'Comanda enviada',
      input: 'Introdueix la MAC',
      ok: "D'acord"
    },
    download: {
      title: "Descarregador d'imatges",
      input: 'Introdueix la URL de la imatge',
      ok: "D'acord",
      disabled: 'La partició /data és només lectura. No es pot descarregar la imatge.',
      uploadbox: 'Deixeu anar el fitxer aquí o feu clic per seleccionar-lo',
      inputfile: "Introduïu el fitxer d'imatge",
      NoISO: 'Cap ISO',
      sha256: 'SHA-256 (opcional)',
      sha256Placeholder: 'Introduïu una suma de verificació SHA-256 de 64 caràcters',
      invalidSHA256: 'SHA-256 ha de ser una cadena hexadecimal de 64 caràcters',
      failed: 'Descàrrega fallida',
      success: 'Descàrrega correcta',
      checksumFailed: 'Descàrrega fallida: ha fallat la verificació SHA-256',
      cancel: 'Cancel·la',
      cancelFailed: 'No sha pogut cancel·lar la descàrrega'
    },
    power: {
      title: 'Alimentació',
      showConfirm: 'Confirmació',
      showConfirmTip: "Les operacions d'alimentació requereixen confirmació",
      reset: 'Reinicia',
      power: 'Encén',
      powerShort: 'Clic curt',
      powerLong: 'Clic llarg',
      resetConfirm: 'Vols realment reiniciar?',
      powerConfirm: 'Vols realment encendre/apagar?',
      okBtn: 'Sí',
      cancelBtn: 'No'
    },
    devices: {
      title: 'Dispositius',
      stale: 'L’estat en directe dels dispositius no està disponible. S’està reconnectant.',
      empty: 'No hi ha cap ranura de càmera ni de micròfon configurada.',
      available: 'Disponible',
      waiting: 'L’amfitrió espera una font',
      hostOpen: 'Amfitrió obert',
      hostIdle: 'Amfitrió inactiu',
      sending: 'S’està enviant des d’aquest navegador',
      black: 'Vídeo negre',
      silence: 'Silenci digital',
      resuming: 'A l’espera de reprendre',
      stop: 'Atura la compartició',
      disconnect: 'Desconnecta',
      takeover: 'Pren el control',
      refused: 'En ús per {{owner}} des de {{source}}',
      connectedSources_one: '{{count}} font connectada',
      connectedSources_other: '{{count}} fonts connectades',
      connectedSources: '{{count}} fonts connectades',
      connection: {
        connecting: 'S’està connectant',
        connected: 'En directe',
        disconnected: 'S’està reconnectant'
      },
      share: {
        camera: 'Comparteix la càmera',
        microphone: 'Comparteix el micròfon',
        usbDevice: 'Comparteix USB'
      },
      permission: {
        denied: 'Bloquejat a la configuració del lloc del navegador',
        prompt: 'El navegador us demanarà permís'
      },
      mic: {
        mute: 'Silencia',
        unmute: 'Deixa de silenciar'
      },
      revoked: {
        released: 'S’ha aturat la compartició',
        lease_expired: 'La concessió ha caducat abans que aquest navegador tornés',
        admin_disconnect: 'Un administrador ha desconnectat totes les fonts',
        slot_removed: 'S’ha eliminat la ranura',
        slot_changed: 'S’ha reconfigurat la ranura',
        taken_over: 'Un administrador ha pres aquesta ranura'
      },
      usb: {
        surrendered: 'El passthrough USB reté el teclat i el ratolí',
        surrenderedDesc:
          'L’amfitrió remot veu el dispositiu importat en lloc del teclat, el ratolí i els mitjans virtuals del NanoKVM. Tornen quan s’atura la sessió.',
        unsupported: 'WebUSB necessita un navegador Chromium sobre HTTPS',
        session: 'S’està reenviant {{device}} ({{mode}})',
        idle: 'Cap sessió de passthrough',
        mode: {
          hybrid: 'híbrid',
          exact: 'exacte'
        }
      }
    },
    settings: {
      title: 'Configuració',
      display: {
        title: 'Pantalla',
        loading: 'Carregant...',
        active: 'EDID actiu',
        activeUnknown:
          "El NanoKVM no ha escrit cap EDID des que es va engegar, així que es desconeix quina identitat veu l'amfitrió.",
        appliedAt: 'Aplicat el {{time}}',
        download: 'Baixa',
        downloadBackup: "Baixa l'anterior",
        preset: 'Perfil de monitor',
        presetPlaceholder: 'Trieu un monitor',
        upload: 'Puja',
        selected: 'EDID seleccionat',
        errors: 'Errors',
        warnings: 'Advertiments',
        info: 'Informació',
        unknownMonitor: 'Monitor desconegut',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Àudio',
        audioNo: 'Sense àudio',
        extensionBlocks: "Blocs d'extensió: {{blocks}}",
        apply: 'Aplica',
        applyTitle: 'Voleu aplicar aquest EDID?',
        before: 'Actual',
        after: 'Nou',
        hdmiNotice:
          "La captura de vídeo s'atura mentre s'escriu l'EDID i es reprèn tota sola en acabar.",
        powerCycleNotice:
          'Cal desconnectar físicament aquest dispositiu del corrent i tornar-lo a connectar perquè el nou EDID tingui efecte.',
        powerCycleUnverified:
          'L’escriptura no s’ha verificat, de manera que el xip de vídeo conserva el que té ara fins que aquest dispositiu es desconnecti físicament del corrent i es torni a connectar.',
        applied: 'EDID aplicat i verificat.',
        applyFailed: "No s'ha pogut aplicar l'EDID.",
        busy: 'El xip de vídeo estava ocupat. Torneu-ho a provar.',
        unsupported: "Aquest dispositiu no permet canviar l'EDID.",
        toolMissing: "Falta l'eina d'EDID en aquest microprogramari.",
        noAudio: "Aquest EDID no anuncia àudio, de manera que l'amfitrió pot deixar d'enviar so.",
        oldVersion: 'Aquest EDID fa servir una versió anterior a la 1.4.',
        interlaced: 'La resolució preferida és entrellaçada.',
        tooLarge:
          'La resolució preferida supera els 1920x1080 a 60 Hz, més del que pot capturar el NanoKVM.',
        recovery: 'Recuperació',
        recoveryNeeded:
          "L'última escriptura no s'ha verificat, de manera que la zona EDID del xip de vídeo es troba en un estat desconegut. Restaura l'EDID de fàbrica per tornar a un estat conegut.",
        recoveryDesc:
          "Torna a escriure un EDID conegut al xip de vídeo quan el que has aplicat ha deixat l'amfitrió sense imatge.",
        restoreFactory: "Restaura l'EDID de fàbrica",
        restoreBackup: "Restaura l'EDID anterior",
        restoreTitle: 'Vols restaurar aquest EDID?',
        restoreFactoryTarget: "L'EDID de fàbrica amb què es lliura el NanoKVM.",
        restoreBackupTarget: 'La còpia més recent, aplicada el {{time}}.',
        restoreNotice:
          "Una restauració s'escriu igual que una aplicació i té les mateixes conseqüències.",
        restored: 'EDID restaurat i verificat.',
        restoreFailed: "No s'ha pogut restaurar l'EDID.",
        mismatchTitle: 'Escrit i rellegit',
        mismatchWritten: 'Escrit',
        mismatchRead: 'Rellegit',
        restoreOkBtn: 'Restaura',
        hardware: 'Maquinari detectat: {{hardware}}',
        hardwareUnknown: 'Desconegut',
        confirmWord: 'APLICA',
        confirmPrompt: "Escriu {{word}} per activar el botó d'aplicar.",
        okBtn: 'Aplica',
        cancelBtn: 'Cancel·la'
      },
      presentation: {
        title: 'Presentació USB',
        loading: 'Carregant...',
        current: 'Presentació USB actual',
        noProfile: 'Cap perfil aplicat',
        linked: 'Funcions enllaçades',
        hostState: 'USB de l’amfitrió',
        hostUnbound: 'Controlador no vinculat',
        hdmiState: 'Entrada HDMI',
        hdmiSignal: 'Hi ha senyal',
        hdmiUnreported: 'Encara no hi ha cap informe de captura',
        endpoints: 'Endpoints',
        fifos: 'Ranures FIFO',
        pending: 'Canvis pendents',
        pendingEdits: 'Canvis d’identitat sense desar',
        pendingProfile: '{{profile}} està seleccionat però no aplicat',
        pendingNone: 'Cap',
        lastApply: 'Darrera aplicació',
        applyFailed: 'Ha fallat a {{profile}} el {{time}}',
        applyClean: 'Cap error registrat',
        lastKnownGood: 'Darrer estat correcte conegut',
        rollbackTarget: 'Objectiu de reversió',
        rollbackNone: 'Cap',
        powerCyclePending:
          'S’ha pres el controlador a l’amfitrió. Apagueu i torneu a engegar l’ordinador connectat per recuperar el dispositiu.',
        rollback: 'Reverteix',
        rollbackTitle: 'Voleu revertir a {{profile}}?',
        rollbackDesc: 'El gadget es torna a enumerar; les funcions USB cauen breument.',
        profile: 'Perfil USB',
        builtIn: 'integrat',
        descriptors: 'descriptors',
        imported: 'importat',
        clone: 'Clona',
        cloneTitle: 'Clona aquest perfil',
        cloneToEdit:
          'Els perfils integrats són només de lectura. Cloneu aquest perfil per editar-ne la identitat.',
        profileName: 'Nom del perfil',
        profileNameHint: 'Lletres minúscules, xifres, punts, guions baixos i guions.',
        import: 'Importa un paquet',
        export: 'Exporta el paquet',
        delete: 'Suprimeix',
        deleteTitle: 'Voleu suprimir aquest perfil?',
        deleteDesc: 'Suprimeix {{profile}} del NanoKVM.',
        identity: 'Identitat USB',
        preset: 'Identitat predefinida',
        presetPlaceholder: 'Copia la identitat d’un dispositiu conegut',
        presetHint:
          'Una predefinició omple el Vendor ID, el Product ID i els dos camps de nom. No aporta cap descriptor.',
        presetSource: 'Identitat tal com consta a {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'Aquest Vendor ID pertany a un altre fabricant',
        productId: 'Product ID',
        bcdUSB: 'Versió d’USB',
        bcdDevice: 'Versió del dispositiu',
        manufacturer: 'Fabricant',
        product: 'Producte',
        serial: 'Número de sèrie',
        configuration: 'Cadena de configuració',
        hidLayout: 'Dispositius HID',
        hidRoleKeyboard: 'Teclat',
        hidRoleRelative: 'Ratolí (relatiu)',
        hidRoleAbsolute: 'Punter (absolut)',
        hidOff: 'No present',
        hidInterface: 'Interfície {{index}}',
        hidBootKeyboardShared:
          'El teclat comparteix una interfície, de manera que ja no ofereix informe en protocol boot. Alguns BIOS i UEFI no el veuran.',
        functions: 'Funcions',
        descriptorAssets: 'Descriptors emmagatzemats: {{count}}',
        endpointUse:
          'IN {{inUse}} en ús, {{inFree}} lliures; OUT {{outUse}} en ús, {{outFree}} lliures',
        preview: 'Valida',
        save: 'Desa',
        apply: 'Aplica',
        applyTitle: 'Voleu aplicar aquest perfil USB?',
        applyDesc: 'El NanoKVM presentarà {{profile}} a l’ordinador connectat.',
        reconnect:
          'El teclat, el ratolí i la resta de funcions USB es desconnecten breument mentre es torna a enllaçar el gadget.',
        applyLinks: 'Enllaça: {{functions}}',
        applyRemoves: 'Elimina: {{functions}}',
        applyNoHid:
          'Després d’aplicar-ho no queda cap funció HID. El teclat i el ratolí deixaran de funcionar.',
        applyRollback: 'Si l’aplicació falla, es torna a {{profile}}.',
        recoveryPowerCycle:
          'Cap HID no sobreviu a aquesta aplicació, així que un amfitrió que deixi de respondre només es pot recuperar apagant-lo i tornant-lo a engegar.',
        recoveryReboot:
          'Una interfície desapareix del dispositiu compost; pot ser que l’amfitrió s’hagi de reiniciar per tornar a vincular la resta.',
        recoveryHdmiReset:
          'Es reconstrueix una funció de vídeo, de manera que la cadena de captura que hi ha al darrere es reinicia.',
        recoveryReconnect:
          'L’amfitrió torna a enumerar el dispositiu; les funcions USB cauen breument.',
        cancel: 'Cancel·la',
        noFunctions: 'Cap funció enllaçada',
        loadFailed: 'No s’han pogut carregar els perfils de presentació',
        operationFailed: 'L’operació de presentació ha fallat'
      },
      passthrough: {
        title: 'Passthrough USB',
        loading: 'Carregant...',
        mode: 'Mode',
        hybrid: 'Híbrid',
        exact: 'Exacte',
        hybridDesc: 'Manté el teclat boot i el ratolí relatiu, per a dispositius compatibles.',
        exactDesc: 'Substitueix totes les funcions USB del NanoKVM pel dispositiu importat.',
        hybridWarning: 'El mode híbrid manté el teclat i el ratolí relatiu',
        hybridWarningDesc:
          'L’emmagatzematge, la xarxa USB i el punter absolut es desconnecten mentre la funció importada està activa.',
        hidWarning: 'Iniciar el passthrough cedeix el teclat, el ratolí i els suports virtuals',
        hidWarningDesc:
          'El NanoKVM només té un controlador de dispositiu USB i el proxy el necessita sencer, així que mentre hi hagi una sessió l’amfitrió remot veurà el dispositiu redirigit en lloc del teclat, el ratolí i els suports virtuals del NanoKVM. Tornen sols en el moment que s’atura la sessió. Aquesta interfície web no es veu afectada, de manera que sempre pots aturar la sessió des d’aquesta pàgina.',
        hidWarningSafeDesc:
          'El NanoKVM només té un controlador de dispositiu USB i el proxy el necessita sencer, així que mentre hi ha una sessió l’amfitrió remot veu el dispositiu redirigit en lloc del teclat, el ratolí i els mitjans virtuals del NanoKVM. Tornen quan s’atura la sessió.',
        isoLabel: 'Permet transferències isòcrones',
        isoHint:
          'Deixa passar càmeres web, micròfons i altres dispositius de flux. Ningú no ha mesurat què aguanta aquest maquinari.',
        isoWarning:
          'El flux isòcron no està provat aquí i pot retenir el teclat i el ratolí fins que aturis la sessió',
        info: {
          title: 'Informació',
          hybrid:
            'El mode híbrid manté disponibles el teclat i el ratolí relatiu. L’emmagatzematge, la xarxa USB i el punter absolut es desconnecten mentre el dispositiu importat és actiu.',
          exact:
            'El mode exacte substitueix totes les funcions USB del NanoKVM pel dispositiu importat. El teclat, el ratolí i els mitjans virtuals tornen sols quan s’atura la sessió.',
          udc: 'El NanoKVM només té un controlador de dispositiu USB i el proxy el necessita sencer: per això les funcions de dalt desapareixen mentre dura una sessió.',
          web: 'Aquesta interfície web no es veu afectada, així que sempre podeu aturar una sessió des d’aquesta pàgina.',
          network:
            'Inicieu el passthrough per Ethernet o Wi-Fi. Iniciar-lo des de la xarxa USB del NanoKVM es rebutja, perquè aquesta connexió desapareixeria.',
          iso: 'Les càmeres web, els micròfons i altres dispositius isòcrons es rebutgen mentre no permeteu les transferències isòcrones. Aquest camí funciona però mai no s’ha mesurat en aquest maquinari: considereu-ne el rendiment desconegut.',
          camera:
            'La càmera i el micròfon del navegador, a Dispositius, continuen sent la manera provada de donar-ne un a la màquina remota.'
        },
        session: 'Sessió',
        activeDesc: 'Hi ha un dispositiu importat i el proxy manté el controlador USB.',
        inactiveDesc:
          'No hi ha cap sessió en marxa. El teclat, el ratolí i els suports virtuals funcionen amb normalitat.',
        device: 'Dispositiu',
        busId: 'ID de bus',
        speed: 'Velocitat',
        exporter: 'Exportador',
        local: 'Importat com a',
        localValue: 'Bus {{bus}}, adreça {{address}}',
        udc: 'Controlador USB',
        pid: 'PID del proxy',
        startedAt: 'Iniciada',
        isoDevice:
          'Aquest dispositiu transmet per punts finals isòcrons, cosa que mai s’ha mesurat en aquest maquinari',
        exporterLabel: 'Adreça de l’exportador',
        exporterHint:
          'L’amfitrió i el port que marca el NanoKVM. Amb el túnel de sota és {{exporter}}.',
        busIdLabel: 'ID de bus a la teva màquina',
        busIdHint: 'El busid que usbip list -l mostra per al dispositiu, per exemple {{example}}.',
        start: 'Inicia el passthrough',
        stop: 'Atura el passthrough',
        startTitle: 'Vols iniciar el passthrough USB?',
        startDevice: 'El NanoKVM importarà {{busId}} des de {{exporter}}.',
        startHid:
          'El teclat USB, el ratolí i els suports virtuals deixen de funcionar mentre duri la sessió i tornen sols quan l’aturis.',
        startIso:
          'Les càmeres web i altres dispositius isòcrons necessiten que activis l’interruptor isòcron abans de començar.',
        startWeb:
          'Aquesta interfície web continua funcionant, així que pots aturar la sessió des d’aquesta pàgina en qualsevol moment.',
        startNetwork:
          'Feu servir aquesta pàgina per Ethernet o Wi-Fi. Iniciar-la des de la xarxa USB del NanoKVM es rebutja perquè aquesta connexió desapareixeria.',
        okBtn: 'Inicia',
        cancelBtn: 'Cancel·la',
        instructions: 'A la teva màquina',
        instructionsDesc:
          'Per disseny no cal instal·lar cap agent client. Executa aquestes ordres estàndard d’usbip a la màquina on hi ha el dispositiu.',
        copyFailed: 'No s’ha pogut copiar. Copia l’ordre manualment.',
        directNote:
          'Sense túnel, l’usbipd ha de ser accessible a la teva xarxa i l’adreça de l’exportador de dalt l’ha d’indicar. L’usbip transporta el dispositiu sense xifrar, així que és preferible el túnel.',
        steps: {
          modprobe: {
            title: 'Carrega el controlador de l’exportador',
            desc: 'usbip-host és el que permet al nucli cedir un dispositiu local. No es carrega per defecte.'
          },
          list: {
            title: 'Troba el dispositiu',
            desc: 'Mostra tots els dispositius locals amb el seu busid i la parella fabricant:producte. Anota el busid del que vulguis.'
          },
          bind: {
            title: 'Vincula’l a usbip',
            desc: 'Treu el dispositiu del seu controlador habitual, de manera que deixa de funcionar en aquesta màquina fins que el desvinculis.'
          },
          serve: {
            title: 'Ofereix-lo',
            desc: 'usbipd es queda en primer pla i espera que el NanoKVM importi el dispositiu.',
            notice:
              'L’usbipd estàndard no té cap opció d’adreça d’escolta i escolta a totes les interfícies. Mantén el port {{port}} tancat al tallafoc i deixa que només hi arribi el túnel de sota.'
          },
          tunnel: {
            title: 'Apunta’l al NanoKVM',
            desc: 'Un túnel invers SSH: el port {{port}} del bucle local del NanoKVM passa a ser l’usbipd d’aquesta màquina. Deixa’l en marxa tota la sessió.'
          },
          exporter: {
            title: 'Fes servir això com a exportador',
            desc: 'Posa-ho al camp de l’exportador de dalt, escriu l’ID de bus i inicia la sessió.'
          },
          unbind: {
            title: 'Torna el dispositiu',
            desc: 'Quan la sessió s’aturi, això retorna el dispositiu al seu controlador habitual en aquesta màquina.'
          }
        }
      },
      mcp: {
        title: 'Servei MCP',
        service: 'Control remot MCP',
        serviceDesc:
          'Permet que clients MCP de confiança controlin el teclat i el ratolí i capturin pantalles',
        securityWarning:
          'Qualsevol persona amb aquesta clau API pot controlar l’amfitrió remot i veure’n la pantalla. Utilitzeu HTTPS i activeu-lo només en xarxes de confiança.',
        endpoint: 'Punt de connexió',
        apiKey: 'Clau API',
        regenerateConfirmTitle: 'Voleu tornar a generar la clau API MCP?',
        regenerateConfirmDesc: 'La clau actual deixarà de funcionar immediatament.',
        enableConfirmTitle: 'Voleu activar el control MCP extern?',
        enableConfirmDesc:
          'En activar MCP, PicoClaw s’aturarà i es tancarà qualsevol sessió activa de PicoClaw.',
        failed: 'L’operació MCP ha fallat',
        copyFailed: 'La còpia ha fallat. Copieu-ho manualment.',
        okBtn: 'Confirma',
        cancelBtn: 'Cancel·la'
      },
      about: {
        title: 'Sobre NanoKVM',
        information: 'Informació',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Versió aplicació',
        applicationTip: 'Versió de la interfície web de NanoKVM',
        image: 'Versió de la imatge',
        imageTip: 'Versió del sistema NanoKVM',
        deviceKey: 'Clau del dispositiu',
        community: 'Comunitat',
        hostname: 'Nom del dispositiu',
        hostnameUpdated: 'Nom actualitzat. Reinicia per aplicar.',
        ipType: {
          Wired: 'Cablejada',
          Wireless: 'Sense fil',
          Other: 'Altra'
        }
      },
      appearance: {
        title: 'Aparença',
        display: 'Pantalla',
        language: 'Idioma',
        languageDesc: "Seleccioneu l'idioma per a la interfície",
        webTitle: 'Títol web',
        webTitleDesc: 'Personalitza el títol de la pàgina',
        menuBar: {
          title: 'Barra de menús',
          mode: 'Mode de visualització',
          modeDesc: 'Mostra la barra de menús a la pantalla',
          modeOff: 'Apagat',
          modeAuto: 'Ocultació automàtica',
          modeAlways: 'Sempre visible',
          keyboardLedStatus: 'Indicadors de bloqueig del teclat',
          keyboardLedStatusDesc:
            'Mostra l’estat de Bloq Num, Bloq Maj i Bloq Despl de l’ordinador remot',
          icons: 'Icones del submenú',
          iconsDesc: 'Mostra les icones del submenú a la barra de menús'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Estat de bloqueig del teclat remot',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Bloq Num',
        numLockShort: 'Num',
        capsLock: 'Bloq Maj',
        capsLockShort: 'Maj',
        scrollLock: 'Bloq Despl',
        scrollLockShort: 'Despl',
        on: 'Activat',
        off: 'Desactivat',
        unknown: 'Desconegut'
      },
      device: {
        title: 'Dispositiu',
        oled: {
          title: 'OLED',
          description: 'Apagar pantalla OLED després de',
          0: 'Mai',
          15: '15 s',
          30: '30 s',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 h'
        },
        ssh: {
          description: 'Activa accés remot per SSH',
          tip: 'Configura una contrasenya segura abans (Compte - Canvia contrasenya)'
        },
        advanced: 'Configuració avançada',
        swap: {
          title: 'Swap',
          disable: 'Desactiva',
          description: 'Defineix la mida del fitxer swap',
          tip: 'Pot reduir la vida útil de la targeta SD!'
        },
        mouseJiggler: {
          title: 'Mou-ratolí automàtic',
          description: 'Evita que el dispositiu remot entri en repòs',
          disable: 'Desactiva',
          absolute: 'Mode absolut',
          relative: 'Mode relatiu'
        },
        mdns: {
          description: 'Activa descobriment mDNS',
          tip: 'Desactiva-ho si no és necessari'
        },
        hdmi: {
          description: 'Activa la sortida HDMI',
          idleTimeoutTitle: "Temps d'espera d'inactivitat de captura",
          idleTimeoutDescription:
            'Atura la captura HDMI després de no detectar espectadors actius durant',
          minutes: 'min'
        },
        autostart: {
          title: "Configuració dels scripts d'inici automàtic",
          description: "Gestioneu els scripts que s'executen automàticament a l'inici del sistema",
          new: 'Nou',
          deleteConfirm: 'Estàs segur que vols eliminar aquest fitxer?',
          yes: 'Sí',
          no: 'No',
          scriptName: "Nom de l'script d'inici automàtic",
          scriptContent: "Contingut de l'script d'inici automàtic",
          settings: 'Configuració'
        },
        hidOnly: 'Mode només HID',
        hidOnlyDesc: "Deixeu d'emular dispositius virtuals, conservant només el control bàsic HID",
        disk: 'Disc virtual',
        diskDesc: 'Munta un disc U virtual al dispositiu remot',
        network: 'Xarxa virtual',
        networkDesc: 'Munta una targeta de xarxa virtual al dispositiu remot',
        networkProtocol: 'Protocol de xarxa',
        networkProtocolDesc: 'NCM per a amfitrions moderns, RNDIS per a Windows antics',
        media: {
          title: 'Ranures de càmera i micròfon',
          desc: 'Declareu els dispositius multimèdia que els navegadors poden omplir. El pressupost de punts finals es comprova en aplicar el perfil USB. En desar, el dispositiu es torna a enumerar i es desconnecta qualsevol navegador connectat.',
          cameras: 'Càmeres',
          microphones: 'Micròfons',
          name: 'Nom',
          namePlaceholder: 'Es mostra a l’amfitrió de destinació',
          addCamera: 'Afegeix una càmera',
          addMicrophone: 'Afegeix un micròfon',
          remove: 'Elimina',
          cameraDefault: 'Càmera NanoKVM {{index}}',
          microphoneDefault: 'Micròfon NanoKVM {{index}}',
          nameRequired: 'Cada ranura necessita un nom.',
          unsupported:
            'Aquest nucli no pot anomenar els dispositius multimèdia, així que els amfitrions mostren el nom per defecte.',
          save: 'Desa les ranures',
          disconnect: 'Desconnecta',
          disconnectAll: 'Desconnecta totes les fonts',
          limit: 'Les ranures de càmera i micròfon no poden sumar més de vuit.',
          failed: 'No s’han pogut actualitzar les ranures multimèdia.'
        },
        reboot: 'Reinicia',
        rebootDesc: 'Segur que vols reiniciar el NanoKVM?',
        okBtn: 'Sí',
        cancelBtn: 'No'
      },
      network: {
        title: 'Xarxa',
        wifi: {
          title: 'Wi-Fi',
          description: 'Configura la xarxa Wi-Fi',
          apMode: "El mode AP està activat; connecta't al Wi-Fi escanejant el codi QR",
          connect: "Connecta't a Wi-Fi",
          connectDesc1: 'Introdueix el SSID i la contrasenya de la xarxa',
          connectDesc2: 'Introdueix la contrasenya per connectar-te a aquesta xarxa',
          disconnect: 'Segur que vols desconnectar la xarxa?',
          failed: 'La connexió ha fallat, torna-ho a provar.',
          ssid: 'Nom',
          password: 'Contrasenya',
          joinBtn: 'Connecta',
          confirmBtn: "D'acord",
          cancelBtn: 'Cancel·la'
        },
        tls: {
          description: 'Activa el protocol HTTPS',
          tip: 'Atenció: Usar HTTPS pot augmentar la latència, sobretot amb vídeo MJPEG.'
        },
        bridge: {
          title: 'Pont de xarxa',
          twoDevices:
            'El teu encaminador veurà el NanoKVM i l’ordinador controlat com dos dispositius separats, cadascun amb la seva pròpia adreça.',
          loading: 'Carregant...',
          state: 'Estat',
          states: {
            disabled: 'Desactivat',
            enabled: 'Activat',
            rolledBack: 'Revertit',
            failed: 'Ha fallat',
            pending: 'En curs'
          },
          uplink: 'Enllaç ascendent',
          ports: 'Ports',
          protocol: 'Protocol del dispositiu',
          up: 'actiu',
          down: 'inactiu',
          noLink: 'sense enllaç',
          enableTitle: 'Vols activar el pont de xarxa?',
          disableTitle: 'Vols desactivar el pont de xarxa?',
          reconnect:
            'La connexió d’administració es desconnectarà i es tornarà a connectar breument mentre es mou l’adreça.',
          rollback: 'Si la verificació falla, es restaura automàticament la configuració anterior.',
          enableBtn: 'Activa',
          disableBtn: 'Desactiva',
          cancelBtn: 'Cancel·la',
          interrupted:
            'La connexió s’ha interromput durant l’aplicació. S’està tornant a comprovar l’estat actual.',
          pendingNotice: 'Un canvi del pont encara està en curs o s’ha interromput abans d’acabar.',
          revert: 'Restaura la configuració anterior',
          rolledBackNotice:
            'L’últim canvi s’ha revertit i s’ha restaurat la configuració anterior.',
          verifyFailed: 'La verificació ha fallat: {{gates}}',
          gates: {
            address: 'adreça',
            gateway: 'passarel·la',
            inbound: 'connexió entrant'
          },
          inboundWeak:
            "La comprovació d'entrada només ha passat perquè el NanoKVM s'ha connectat a si mateix. Això demostra que el servei web escolta i és accessible localment, no que hi arribi una petició des de la xarxa.",
          noCarrier:
            'No hi ha enllaç a {{port}}. El pont no té cap camí cap a la xarxa fins que s’hi connecti un cable.',
          loop: 'El router també s’està aprenent a {{port}}, de manera que aquest port és un segon camí cap a la mateixa xarxa. L’spanning tree està desactivat, així que res d’aquí no trencarà el bucle: desconnecteu un dels dos camins.',
          failedNotice:
            'No s’ha pogut desfer l’últim canvi. Pot ser que només es pugui accedir al NanoKVM per l’AP Wi-Fi o una consola sèrie.'
        },
        dns: {
          title: 'DNS',
          description: 'Configura els servidors DNS per a NanoKVM',
          mode: 'Mode',
          dhcp: 'DHCP',
          manual: 'Manual',
          add: 'Afegeix DNS',
          save: 'Desa',
          invalid: 'Introdueix una adreça IP vàlida',
          noDhcp: 'No hi ha cap DNS DHCP disponible actualment',
          saved: 'Configuració DNS desada',
          saveFailed: "No s'ha pogut desar la configuració DNS",
          unsaved: 'Canvis no desats',
          maxServers: 'Es permeten com a màxim {{count}} servidors DNS',
          dnsServers: 'Servidors DNS',
          dhcpServersDescription: "Els servidors DNS s'obtenen automàticament via DHCP",
          manualServersDescription: 'Els servidors DNS es poden editar manualment',
          networkDetails: 'Detalls de xarxa',
          interface: 'Interfície',
          ipAddress: 'Adreça IP',
          subnetMask: 'Màscara de subxarxa',
          router: 'Encaminador',
          none: 'Cap'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'Servidor VNC',
        description:
          'Permet que qualsevol client VNC vegi la pantalla remota i faci servir el teclat i el ratolí, entrant amb el vostre compte del NanoKVM',
        port: 'Port',
        portDescription: 'Connecteu-vos a aquest port de l’adreça del NanoKVM'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Optimització de memòria',
          tip: 'Quan es supera el límit de memòria, es fa una neteja més agressiva. Recomanat: 75MB si uses Tailscale. Requereix reiniciar Tailscale.'
        },
        swap: {
          title: 'Intercanvi de memòria',
          tip: "Si els problemes persisteixen després d'activar l'optimització de memòria, proveu d'habilitar la memòria d'intercanvi. Això estableix la mida del fitxer d'intercanvi a 256MB per defecte, que es pot ajustar a \"Configuració > Dispositiu\"."
        },
        restart: 'Reiniciar Tailscale?',
        stop: 'Aturar Tailscale?',
        stopDesc: 'Tanca la sessió de Tailscale i desactiva l’inici automàtic en arrencar.',
        loading: 'Carregant...',
        notInstall: 'Tailscale no instal·lat! Instal·la-ho.',
        install: 'Instal·la',
        installing: 'Instal·lant',
        failed: 'Error en la instal·lació',
        retry: 'Actualitza i torna-ho a provar. O instal·la manualment',
        download: 'Descarrega el',
        package: "paquet d'instal·lació",
        unzip: 'i descomprimeix-lo',
        upTailscale: 'Puja tailscale al directori /usr/bin/ del NanoKVM',
        upTailscaled: 'Puja tailscaled al directori /usr/sbin/ del NanoKVM',
        refresh: 'Actualitza la pàgina',
        notRunning: "Tailscale no s'està executant. Si us plau, inicieu-lo per continuar.",
        run: 'Comença',
        notLogin: 'El dispositiu no està vinculat. Inicia sessió per vincular-lo.',
        urlPeriod: 'Aquesta URL és vàlida durant 10 minuts',
        login: 'Inicia sessió',
        loginSuccess: 'Sessió iniciada correctament',
        enable: 'Activa Tailscale',
        deviceName: 'Nom del dispositiu',
        deviceIP: 'IP del dispositiu',
        account: 'Compte',
        logout: 'Tanca sessió',
        logoutDesc: 'Segur que vols tancar sessió?',
        uninstall: 'Desinstal·la Tailscale',
        uninstallDesc: 'Esteu segur que voleu desinstal·lar Tailscale?',
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
        loading: 'Carregant...',
        notInstall: 'No instal·lat',
        notConfigured: 'Sense configurar',
        stopped: 'Aturat',
        running: 'En execució',
        connected: 'Connectat',
        error: 'Error',
        atBoot: 's’inicia en arrencar',
        notAtBoot: 'no s’inicia en arrencar',
        arguments: 'Arguments',
        argumentsTip: 'Arguments de línia d’ordres que es passen al servei en iniciar-se.',
        env: 'Variables d’entorn',
        envKey: 'Nom',
        envValue: 'Valor',
        envAdd: 'Afegeix una variable',
        envRemove: 'Elimina',
        configured: 'Configurada',
        save: 'Desa',
        saved: 'Configuració desada',
        start: 'Inicia',
        stop: 'Atura',
        restart: 'Reinicia',
        logs: 'Registres',
        logsEmpty: 'Encara no hi ha registres',
        refresh: 'Actualitza',
        binary: 'Binari',
        binaryShipped: 'Inclòs al firmware',
        binaryCustom: 'Binari personalitzat',
        binaryUpload: 'Puja un binari',
        binaryRevert: 'Restaura el binari inclòs',
        binaryRevertDesc: 'Voleu suprimir el binari pujat i restaurar el que inclou el firmware?',
        serverWarning: 'Un servidor sense restriccions actua com a proxy obert',
        noHealthSignal:
          "Aquest servei no informa de cap senyal d'estat, així que el NanoKVM només sap que el procés s'executa, no si el túnel està connectat.",
        memoryWarning: 'Executar diversos serveis d’accés remot alhora pot esgotar la memòria',
        resources: 'Recursos',
        memory: {
          title: 'Límit de memòria',
          description:
            'Limita l’heap de Go del newt a {{limit}} MiB a partir del seu proper reinici. És el seu límit, no el de Tailscale; desactivat s’aplica el valor per defecte de Go, amb GOGC=50 en tots dos casos.',
          noRuntime:
            'El wstunnel és Rust: no hi ha recol·lector d’escombraries ni límit d’heap per fixar, i els seus fils de treball ja segueixen l’única CPU del dispositiu.',
          notApplicable: 'No aplicable'
        },
        swap: {
          title: 'Fitxer d’intercanvi',
          description:
            'Afegeix un fitxer d’intercanvi de 256 MB a la targeta SD. És de tot el sistema: el mateix intercanvi serveix el Tailscale, el servidor KVM i tota la resta del dispositiu.'
        },
        okBtn: 'Sí',
        cancelBtn: 'No'
      },
      update: {
        title: 'Comprova actualitzacions',
        queryFailed: 'Error en obtenir la versió',
        updateFailed: 'Error en actualitzar. Torna-ho a intentar.',
        isLatest: 'Ja tens la darrera versió.',
        rebooting:
          "S'està instal·lant el nucli nou i el dispositiu es reinicia. Pot trigar uns minuts; no l'apagueu.",
        kernelUpdate:
          'Aquesta actualització instal·la el nucli {{version}}. El dispositiu es reiniciarà i tornarà sol al nucli actual si el nou no arrenca.',
        rolledBack:
          'El nucli {{version}} no ha arrencat i el dispositiu ha tornat al nucli anterior.',
        available: 'Hi ha una actualització disponible. Vols actualitzar ara?',
        updating: 'Actualitzant... espera',
        confirm: 'Confirma',
        cancel: 'Cancel·la',
        preview: 'Versió de prova',
        previewDesc: 'Prova noves funcions abans que ningú',
        previewTip: 'Compte: aquestes versions poden tenir errors o funcions inacabades!',
        customServer: {
          title: 'Servidor d’actualitzacions personalitzat',
          desc: 'Cerca i baixa actualitzacions en línia des d’un servidor especificat',
          invalidUrl:
            'Introduïu un directori de servidor HTTP o HTTPS vàlid, sense paràmetres de consulta, fragments ni latest.json.',
          loadFailed: 'No s’ha pogut carregar la configuració del servidor d’actualitzacions.',
          saveFailed: 'No s’ha pogut desar la configuració del servidor d’actualitzacions.',
          saved: 'S’ha desat la configuració del servidor d’actualitzacions.',
          save: 'Desa',
          confirmTitle: 'Voleu utilitzar un servidor d’actualitzacions personalitzat?',
          confirmDesc:
            'SHA-512 només comprova que el paquet coincideixi amb el manifest proporcionat per aquest servidor. Això no demostra que el paquet sigui una versió oficial de NanoKVM. Un servidor defectuós o maliciós pot deixar el dispositiu inutilitzable, provocar la pèrdua de dades o comprometre el sistema.',
          confirm: 'Utilitza’l igualment',
          previewDisabled:
            'Les actualitzacions de previsualització no estan disponibles mentre hi hagi activat un servidor d’actualitzacions personalitzat.'
        },
        offline: {
          title: 'Actualitzacions fora de línia',
          desc: "Actualització mitjançant el paquet d'instal·lació local",
          upload: 'Puja',
          checksumPlaceholder: 'Suma de verificació SHA-256 (opcional)',
          invalidChecksum:
            'La suma de verificació SHA-256 ha de contenir 64 caràcters hexadecimals.',
          checksumMismatch:
            'La verificació SHA-256 ha fallat. És possible que el paquet estigui malmès.',
          invalidName: 'Format de nom de fitxer no vàlid. Baixeu-lo des de les versions de GitHub.',
          updateFailed: 'Error en actualitzar. Torna-ho a intentar.'
        }
      },
      account: {
        title: 'Compte',
        webAccount: 'Nom del compte web',
        role: 'Rol',
        roles: {
          admin: 'Administrador',
          user: 'Usuari'
        },
        password: 'Contrasenya',
        updateBtn: 'Canvia',
        logoutBtn: 'Tanca sessió',
        logoutDesc: 'Segur que vols tancar sessió?',
        okBtn: 'Sí',
        cancelBtn: 'No',
        users: {
          title: 'Usuaris',
          create: 'Crea un usuari',
          enabled: 'Activat',
          disabled: 'Desactivat',
          deviceOwner: 'Propietari del dispositiu',
          resetPassword: 'Restableix la contrasenya',
          delete: 'Elimina',
          deleteConfirm: 'Voleu eliminar aquest usuari i revocar totes les seves sessions?',
          created: 'Usuari creat',
          deleted: 'Usuari eliminat',
          passwordUpdated: 'Contrasenya actualitzada',
          loadFailed: 'No s’han pogut carregar els usuaris',
          saveFailed: 'No s’ha pogut desar l’usuari',
          deleteFailed: 'No s’ha pogut eliminar l’usuari'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Assistent',
      empty: 'Obriu el tauler i inicieu una tasca per començar.',
      inputPlaceholder: 'Descriu què vols que faci el PicoClaw',
      newConversation: 'Nova conversa',
      processing: "S'està processant...",
      agent: {
        defaultTitle: 'Assistent general',
        defaultDescription: 'Ajuda general de xat, cerca i espai de treball.',
        kvmTitle: 'Control remot',
        kvmDescription: "Opera l'amfitrió remot mitjançant NanoKVM.",
        switched: "Rol d'agent canviat",
        switchFailed: "No s'ha pogut canviar la funció d'agent"
      },
      send: 'Envia',
      cancel: 'Cancel·la',
      status: {
        connecting: "S'està connectant a la passarel·la...",
        connected: 'Sessió de PicoClaw connectada',
        disconnected: 'Sessió de PicoClaw tancada',
        stopped: "S'ha enviat la sol·licitud d'aturada",
        runtimeStarted: "Temps d'execució de PicoClaw iniciat",
        runtimeStartFailed: "No s'ha pogut iniciar el temps d'execució de PicoClaw",
        runtimeStopped: "Temps d'execució de PicoClaw aturat",
        runtimeStopFailed: "No s'ha pogut aturar el temps d'execució de PicoClaw",
        controlSwitchedToMCP: 'El control ha canviat al servei MCP extern'
      },
      connection: {
        runtime: {
          checking: 'Comprovació',
          restoring: 'Restoring PicoClaw',
          ready: "Temps d'execució a punt",
          stopped: "El temps d'execució s'ha aturat",
          blockedByMCP: 'El control MCP extern està actiu',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: "Temps d'execució no disponible",
          configError: 'Error de configuració'
        },
        transport: {
          connecting: 'En connexió',
          connected: 'Connectat',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Inactiu',
          busy: 'Ocupat'
        }
      },
      message: {
        toolAction: 'Acció',
        observation: 'Observació',
        screenshot: 'Captura de pantalla'
      },
      overlay: {
        locked: "PicoClaw està controlant el dispositiu. L'entrada manual està en pausa."
      },
      control: {
        picoclaw: 'Control del dispositiu: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Control del dispositiu: MCP extern',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Control del dispositiu: desactivat',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Concedeix control',
        release: 'Allibera',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'Control de PicoClaw concedit',
        released: 'Control de PicoClaw alliberat',
        grantFailed: "No s'ha pogut concedir el control a PicoClaw",
        releaseFailed: "No s'ha pogut alliberar el control de PicoClaw",
        grantConfirmTitle: 'Vols canviar el control del dispositiu a PicoClaw?',
        grantConfirmDesc: "Les escriptures del dispositiu MCP extern s'interrompran."
      },
      install: {
        install: 'Instal·la PicoClaw',
        installing: 'Instal·lant PicoClaw',
        success: 'PicoClaw instal·lat correctament',
        failed: "No s'ha pogut instal·lar PicoClaw",
        uninstalling: "S'està desinstal·lant el temps d'execució...",
        uninstalled: "El temps d'execució s'ha desinstal·lat correctament.",
        uninstallFailed: 'La desinstal·lació ha fallat.',
        requiredTitle: 'PicoClaw no està instal·lat',
        requiredDescription: "Instal·leu PicoClaw abans d'iniciar el temps d'execució de PicoClaw.",
        progressDescription: "PicoClaw s'està baixant i instal·lant.",
        stages: {
          preparing: 'Preparant',
          downloading: "S'està baixant",
          extracting: 'Extracció',
          verifying: 'Verificant',
          installing: 'Instal·lant',
          installed: 'Instal·lat',
          install_timeout: 'Temps esgotat',
          install_failed: 'Ha fallat'
        }
      },
      model: {
        requiredTitle: 'La configuració del model és necessària',
        requiredDescription: "Configura el model PicoClaw abans d'utilitzar el xat PicoClaw.",
        docsTitle: 'Guia de configuració',
        docsDesc: 'Models i protocols compatibles',
        menuLabel: 'Configura el model',
        modelIdentifier: 'Identificador del model',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'Clau API',
        apiKeyPlaceholder: 'Introduïu la clau API del model',
        save: 'Desa',
        saving: 'Desa',
        saved: "S'ha desat la configuració del model",
        saveFailed: "No s'ha pogut desar la configuració del model",
        invalid: "Cal indicar l'identificador del model, l'API Base URL i la clau API"
      },
      uninstall: {
        menuLabel: 'Desinstal·la',
        confirmTitle: 'Desinstal·la PicoClaw',
        confirmContent:
          "Esteu segur que voleu desinstal·lar PicoClaw? Això suprimirà l'executable i tots els fitxers de configuració.",
        confirmOk: 'Desinstal·la',
        confirmCancel: 'Cancel·la'
      },
      history: {
        title: 'Historial',
        loading: 'Carregant sessions...',
        emptyTitle: 'Encara no hi ha historial',
        emptyDescription: 'Les sessions anteriors de PicoClaw apareixeran aquí.',
        loadFailed: "No s'ha pogut carregar l'historial de sessions",
        deleteFailed: "No s'ha pogut suprimir la sessió",
        deleteConfirmTitle: 'Suprimeix la sessió',
        deleteConfirmContent: 'Esteu segur que voleu suprimir "{{title}}"?',
        deleteConfirmOk: 'Esborra',
        deleteConfirmCancel: 'Cancel·la',
        messageCount_one: '{{count}} missatge',
        messageCount_other: '{{count}} missatges',
        messageCount: '{{count}} missatges'
      },
      config: {
        startRuntime: 'Inici PicoClaw',
        stopRuntime: 'Atura PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Voleu canviar el control a PicoClaw?',
        enableConfirmDesc: 'En iniciar PicoClaw es desactivarà el servei MCP extern.',
        enableConfirmOk: 'Inicia PicoClaw',
        enableConfirmCancel: 'Cancel·la',
        title: 'Inici PicoClaw',
        description: "Inicieu el temps d'execució per començar a utilitzar l'assistent PicoClaw.",
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Hi ha hagut un error',
      refresh: 'Actualitza'
    },
    fullscreen: {
      toggle: 'Pantalla completa'
    },
    menu: {
      collapse: 'Amaga menú',
      expand: 'Mostra menú'
    }
  }
};

export default ca;
