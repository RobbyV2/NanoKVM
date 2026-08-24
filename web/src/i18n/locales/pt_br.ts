const pt_br = {
  translation: {
    head: {
      desktop: 'Área de Trabalho Remota',
      login: 'Login',
      changePassword: 'Mudar Senha',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Login',
      placeholderUsername: 'Nome de usuário',
      placeholderPassword: 'Senha',
      placeholderCurrentPassword: 'Senha atual',
      placeholderPassword2: 'Por favor, digite a senha novamente',
      noEmptyUsername: 'Nome de usuário é obrigatório',
      noEmptyPassword: 'Senha é obrigatória',
      passwordLength: 'A senha deve ter entre 8 e 72 caracteres',
      noAccount:
        'Falha ao obter informações do usuário, por favor atualize a página ou redefina a senha',
      invalidUser: 'Nome de usuário ou senha inválidos',
      locked: 'Muitos logins, tente novamente mais tarde',
      globalLocked: 'Sistema sob proteção, tente novamente mais tarde',
      error: 'Erro inesperado',
      invalidCurrentPassword: 'A senha atual está incorreta',
      changePassword: 'Mudar Senha',
      changePasswordDesc: 'Para a segurança do seu dispositivo, por favor, mude a senha!',
      differentPassword: 'Senhas não conferem',
      illegalUsername: 'Nome de usuário contém caracteres inválidos',
      illegalPassword: 'Senha contém caracteres inválidos',
      forgetPassword: 'Esqueci a senha',
      ok: 'Ok',
      cancel: 'Cancelar',
      loginButtonText: 'Login',
      tips: {
        reset1:
          'Para redefinir as senhas, pressione e segure o botão BOOT no NanoKVM por 10 segundos.',
        reset2: 'Para etapas detalhadas, por favor, consulte este documento:',
        reset3: 'Conta padrão da Web:',
        reset4: 'Conta padrão SSH:',
        change1: 'Por favor, note que esta ação irá alterar as seguintes senhas:',
        change2: 'Senha de login da Web',
        change3: 'Senha root do sistema (senha de login SSH)',
        change4: 'Para redefinir as senhas, pressione e segure o botão BOOT no NanoKVM.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Configurar Wi-Fi para o NanoKVM',
      success: 'Por favor, verifique o status da rede do NanoKVM e visite o novo endereço IP.',
      failed: 'Operação falhou, por favor, tente novamente.',
      invalidMode:
        'O modo atual não suporta configuração de rede. Vá para o seu dispositivo e ative o modo de configuração Wi-Fi.',
      confirmBtn: 'Ok',
      finishBtn: 'Finalizado',
      ap: {
        authTitle: 'Autenticação necessária',
        authDescription: 'Por favor, digite a senha AP para continuar',
        authFailed: 'Senha AP inválida',
        passPlaceholder: 'AP senha',
        verifyBtn: 'Verificar'
      }
    },
    screen: {
      scale: 'Escala',
      title: 'Tela',
      video: 'Modo de Vídeo',
      videoDirectTips: 'Ative HTTPS em "Configurações > Dispositivo" para usar este modo',
      resolution: 'Resolução',
      auto: 'Automático',
      autoTips:
        'Rasgos na tela ou desvio do mouse podem ocorrer em resoluções específicas. Considere ajustar a resolução do host remoto ou desativar o modo automático.',
      fps: 'FPS',
      customizeFps: 'Personalizar',
      quality: 'Qualidade',
      qualityLossless: 'Sem perdas',
      qualityHigh: 'Alta',
      qualityMedium: 'Média',
      qualityLow: 'Baixa',
      frameDetect: 'Detecção de Quadros',
      frameDetectTip:
        'Calcular a diferença entre os quadros. Parar a transmissão de vídeo quando nenhuma alteração for detectada na tela do host remoto.',
      resetHdmi: 'Redefinir HDMI',
      mixedH264: {
        title: 'Conflito de transmissão H.264',
        description:
          'H.264 Direct e H.264 WebRTC estão sendo usados ao mesmo tempo. Isso pode causar rasgos na tela ou vídeo corrompido. Use apenas um modo H.264.'
      },
      webrtcConnectionFailed: {
        title: 'Falha na conexão WebRTC',
        description: 'Verifique a conexão de rede ou alterne o modo de vídeo.'
      },
      captureStatus: {
        hdmiError: 'Erro na imagem HDMI',
        unsupportedResolution: 'A resolução atual não é compatível',
        retrieving: 'Obtendo tela...',
        changingResolution: 'Alterando resolução...',
        updateFailed: 'A tela não pode ser atualizada agora',
        videoError: 'Erro na exibição de vídeo',
        noHdmi: 'Nenhum sinal HDMI detectado',
        unavailable: 'A tela não pode ser exibida agora'
      }
    },
    keyboard: {
      title: 'Teclado',
      paste: 'Colar',
      tips: 'Apenas letras e símbolos de teclado padrão são suportados',
      placeholder: 'Por favor, digite',
      submit: 'Enviar',
      virtual: 'Teclado',
      readClipboard: 'Ler da área de transferência',
      clipboardPermissionDenied:
        'Permissão da área de transferência negada. Permita o acesso à área de transferência no seu navegador.',
      clipboardReadError: 'Falha ao ler a área de transferência',
      dropdownEnglish: 'Inglês',
      dropdownGerman: 'Alemão',
      dropdownFrench: 'Francês',
      dropdownRussian: 'Russo',
      shortcut: {
        title: 'Atalhos',
        custom: 'Personalizado',
        capture: 'Clique aqui para capturar o atalho',
        clear: 'Limpar',
        save: 'Salvar',
        captureTips:
          'Capturar teclas do sistema (como a tecla Windows) requer permissão de tela cheia.',
        enterFullScreen: 'Alternar modo de tela cheia.'
      },
      leaderKey: {
        title: 'Tecla Leader',
        desc: 'Ignore as restrições do navegador e envie atalhos do sistema diretamente para o host remoto.',
        howToUse: 'Como usar',
        simultaneous: {
          title: 'Modo Simultâneo',
          desc1: 'Pressione e segure a tecla Leader e depois pressione o atalho.',
          desc2: 'Intuitivo, mas pode entrar em conflito com atalhos do sistema.'
        },
        sequential: {
          title: 'Modo Sequencial',
          desc1:
            'Pressione a tecla Leader → pressione o atalho em sequência → pressione a tecla Leader novamente.',
          desc2: 'Requer mais etapas, mas evita completamente conflitos de sistema.'
        },
        enable: 'Habilitar tecla Leader',
        tip: 'Quando atribuída como tecla Leader, esta tecla funciona apenas como gatilho de atalho e perde seu comportamento padrão.',
        placeholder: 'Pressione a tecla Leader',
        shiftRight: 'Shift direito',
        ctrlRight: 'Ctrl direito',
        metaRight: 'Win direito',
        submit: 'Enviar',
        recorder: {
          rec: 'REC',
          activate: 'Ativar teclas',
          input: 'Por favor, pressione o atalho...'
        }
      }
    },
    mouse: {
      title: 'Mouse',
      cursor: 'Estilo do cursor',
      default: 'Cursor padrão',
      pointer: 'Cursor de ponteiro',
      cell: 'Cursor de célula',
      text: 'Cursor de texto',
      grab: 'Cursor de arrastar',
      hide: 'Ocultar cursor',
      mode: 'Modo do mouse',
      absolute: 'Modo absoluto',
      relative: 'Modo relativo',
      direction: 'Direção da roda de rolagem',
      scrollUp: 'Role para cima',
      scrollDown: 'Role para baixo',
      speed: 'Velocidade da roda de rolagem',
      fast: 'Rápido',
      slow: 'Lento',
      requestPointer:
        'Usando modo relativo. Por favor, clique na área de trabalho para obter o ponteiro do mouse.',
      resetHid: 'Redefinir HID',
      hidOnly: {
        title: 'Modo somente HID',
        desc: 'Se o seu mouse e teclado pararem de responder e a redefinição de HID não ajudar, pode ser um problema de compatibilidade entre o NanoKVM e o dispositivo. Tente habilitar o modo Somente-HID para melhor compatibilidade.',
        tip1: 'Habilitar o modo Somente-HID irá desmontar o U-disk virtual e a rede virtual',
        tip2: 'No modo Somente-HID, a montagem de imagem está desativada',
        tip3: 'NanoKVM será reiniciado automaticamente após a troca de modos',
        enable: 'Habilitar modo Somente-HID',
        disable: 'Desabilitar modo Somente-HID'
      }
    },
    image: {
      title: 'Imagens',
      loading: 'Carregando...',
      empty: 'Nada Encontrado',
      mountMode: 'Modo de montagem',
      mountFailed: 'Falha na Montagem',
      mountDesc:
        'Em alguns sistemas, é necessário ejetar o disco virtual no host remoto antes de montar a imagem.',
      unmountFailed: 'Falha na desmontagem',
      unmountDesc:
        'Em alguns sistemas, é necessário ejetar manualmente do host remoto antes de desmontar a imagem.',
      refresh: 'Atualizar a lista de imagens',
      attention: 'Atenção',
      deleteConfirm: 'Tem certeza que deseja excluir esta imagem?',
      okBtn: 'Sim',
      cancelBtn: 'Não',
      tips: {
        title: 'Como fazer upload',
        usb1: 'Conecte o NanoKVM ao seu computador via USB.',
        usb2: 'Certifique-se de que o disco virtual está montado (Configurações - Disco Virtual).',
        usb3: 'Abra o disco virtual no seu computador e copie o arquivo de imagem para o diretório raiz do disco virtual.',
        scp1: 'Certifique-se de que o NanoKVM e seu computador estão na mesma rede local.',
        scp2: 'Abra um terminal no seu computador e use o comando SCP para fazer upload do arquivo de imagem para o diretório /data no NanoKVM.',
        scp3: 'Exemplo: scp seu-caminho-da-imagem root@seu-ip-nanokvm:/data',
        tfCard: 'Cartão TF',
        tf1: 'Este método é suportado em sistemas Linux',
        tf2: 'Remova o cartão TF do NanoKVM (para a versão FULL, desmonte a caixa primeiro).',
        tf3: 'Insira o cartão TF em um leitor de cartão e conecte-o ao seu computador.',
        tf4: 'Copie o arquivo de imagem para o diretório /data no cartão TF.',
        tf5: 'Insira o cartão TF no NanoKVM.'
      }
    },
    script: {
      title: 'Scripts',
      upload: 'Upload',
      run: 'Executar',
      runBackground: 'Executar em segundo plano',
      runFailed: 'Falha na execução',
      attention: 'Atenção',
      delDesc: 'Tem certeza de que deseja excluir este arquivo?',
      confirm: 'Sim',
      cancel: 'Não',
      delete: 'Excluir',
      close: 'Fechar'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'Terminal NanoKVM',
      serial: 'Terminal de Porta Serial',
      serialPort: 'Porta Serial',
      serialPortPlaceholder: 'Por favor, digite a porta serial',
      baudrate: 'Taxa de transmissão',
      parity: 'Paridade',
      parityNone: 'Nenhum',
      parityEven: 'Par',
      parityOdd: 'Ímpar',
      flowControl: 'Controle de fluxo',
      flowControlNone: 'Nenhum',
      flowControlSoft: 'Software',
      flowControlHard: 'Hardware',
      dataBits: 'Bits de dados',
      stopBits: 'Bits de parada',
      confirm: 'Ok'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Enviando comando...',
      sent: 'Comando enviado',
      input: 'Por favor, digite o MAC',
      ok: 'Ok'
    },
    download: {
      title: 'Baixador de Imagens',
      input: 'Por favor, digite uma URL de imagem remota',
      ok: 'Ok',
      disabled: 'A partição /data é RO, então não podemos baixar a imagem',
      uploadbox: 'Solte o arquivo aqui ou clique para selecionar',
      inputfile: 'Por favor insira o arquivo de imagem',
      NoISO: 'Sem ISO',
      sha256: 'SHA-256 (opcional)',
      sha256Placeholder: 'Digite um checksum SHA-256 de 64 caracteres',
      invalidSHA256: 'SHA-256 deve ser uma sequência hexadecimal de 64 caracteres',
      failed: 'Falha no download',
      success: 'Download concluído',
      checksumFailed: 'Falha no download: a verificação SHA-256 falhou',
      cancel: 'Cancelar',
      cancelFailed: 'Falha ao cancelar o download'
    },
    power: {
      title: 'Energia',
      showConfirm: 'Confirmação',
      showConfirmTip: 'Operações de energia requerem uma confirmação extra',
      reset: 'Redefinir',
      power: 'Energia',
      powerShort: 'Energia (clique curto)',
      powerLong: 'Energia (clique longo)',
      resetConfirm: 'Prosseguir com a operação de redefinição?',
      powerConfirm: 'Prosseguir com a operação de energia?',
      okBtn: 'Sim',
      cancelBtn: 'Não'
    },
    devices: {
      title: 'Dispositivos',
      stale: 'O estado ao vivo dos dispositivos está indisponível. Reconectando.',
      empty:
        'Nenhum slot de câmera ou microfone está configurado. Adicione um em Configurações, Dispositivo.',
      available: 'Disponível',
      waiting: 'O host está esperando uma fonte',
      hostOpen: 'Host aberto',
      hostIdle: 'Host ocioso',
      sending: 'Enviando deste navegador',
      black: 'Vídeo preto',
      silence: 'Silêncio digital',
      resuming: 'Aguardando para retomar',
      stop: 'Parar de compartilhar',
      disconnect: 'Desconectar',
      takeover: 'Assumir',
      refused: 'Em uso por {{owner}} a partir de {{source}}',
      connectedSources_one: '{{count}} fonte conectada',
      connectedSources_other: '{{count}} fontes conectadas',
      connectedSources: '{{count}} fontes conectadas',
      connection: {
        connecting: 'Conectando',
        connected: 'Ao vivo',
        disconnected: 'Reconectando'
      },
      share: {
        camera: 'Compartilhar câmera',
        microphone: 'Compartilhar microfone',
        usbDevice: 'Compartilhar USB'
      },
      permission: {
        denied: 'Bloqueado nas configurações do site no seu navegador',
        prompt: 'O navegador vai pedir permissão',
        insecure:
          'Esta página não é servida por HTTPS, então o navegador bloqueia este dispositivo. Ative o HTTPS em Configurações, Rede.'
      },
      capture: {
        unsupported: 'Este navegador não consegue capturar áudio nem vídeo',
        camera: 'Este navegador não consegue codificar os quadros da câmera',
        microphone: 'Este navegador não consegue processar o áudio do microfone'
      },
      mic: {
        mute: 'Silenciar',
        unmute: 'Reativar som'
      },
      revoked: {
        released: 'O compartilhamento foi interrompido',
        lease_expired: 'A concessão expirou antes de este navegador voltar',
        admin_disconnect: 'Um administrador desconectou todas as fontes',
        slot_removed: 'O slot foi removido',
        slot_changed: 'O slot foi reconfigurado',
        taken_over: 'Um administrador assumiu este slot'
      },
      usb: {
        surrendered: 'O passthrough USB está segurando o teclado e o mouse',
        surrenderedDesc:
          'O host remoto vê o dispositivo importado no lugar do teclado, do mouse e das mídias virtuais do NanoKVM. Eles voltam quando a sessão termina.',
        unsupported: 'O WebUSB precisa de um navegador Chromium',
        insecure:
          'Esta página não é servida por HTTPS, então o navegador bloqueia o WebUSB. Ative o HTTPS em Configurações, Rede.',
        session: 'Encaminhando {{device}} ({{mode}})',
        idle: 'Nenhuma sessão de passthrough',
        mode: {
          hybrid: 'híbrido',
          exact: 'exato'
        }
      }
    },
    settings: {
      title: 'Configurações',
      display: {
        title: 'Tela',
        loading: 'Carregando...',
        active: 'EDID ativo',
        activeUnknown:
          'O NanoKVM não gravou nenhum EDID desde que iniciou, portanto a identidade vista pelo host é desconhecida.',
        appliedAt: 'Aplicado em {{time}}',
        download: 'Baixar',
        downloadBackup: 'Baixar o anterior',
        preset: 'Predefinição de monitor',
        presetPlaceholder: 'Selecione um monitor',
        upload: 'Enviar',
        selected: 'EDID selecionado',
        errors: 'Erros',
        warnings: 'Avisos',
        info: 'Informações',
        unknownMonitor: 'Monitor desconhecido',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Áudio',
        audioNo: 'Sem áudio',
        extensionBlocks: 'Blocos de extensão: {{blocks}}',
        apply: 'Aplicar',
        applyTitle: 'Aplicar este EDID?',
        before: 'Atual',
        after: 'Novo',
        hdmiNotice: 'A captura de vídeo para enquanto o EDID é gravado e volta sozinha em seguida.',
        powerCycleNotice:
          'Este dispositivo precisa ser fisicamente desconectado da energia e ligado de novo para que o novo EDID tenha efeito.',
        powerCycleUnverified:
          'A gravação não foi verificada, então o chip de vídeo mantém o que tem agora até este dispositivo ser fisicamente desconectado da energia e ligado de novo.',
        applied: 'EDID aplicado e verificado.',
        applyFailed: 'Falha ao aplicar o EDID.',
        busy: 'O chip de vídeo estava ocupado. Tente de novo.',
        unsupported: 'Este dispositivo não permite alterar o EDID.',
        toolMissing: 'A ferramenta de EDID não existe neste firmware.',
        noAudio: 'Este EDID não anuncia áudio, então o host pode parar de enviar som.',
        oldVersion: 'Este EDID usa uma versão anterior à 1.4.',
        interlaced: 'A resolução preferida é entrelaçada.',
        tooLarge:
          'A resolução preferida é maior que 1920x1080 a 60 Hz, acima do que o NanoKVM consegue capturar.',
        recovery: 'Recuperação',
        recoveryNeeded:
          'A última gravação não foi verificada, então a área de EDID do chip de vídeo está em um estado desconhecido. Restaure o EDID de fábrica para voltar a um estado conhecido.',
        recoveryDesc:
          'Grava um EDID conhecido de volta no chip de vídeo quando o EDID aplicado deixou o host sem imagem.',
        restoreFactory: 'Restaurar EDID de fábrica',
        restoreBackup: 'Restaurar EDID anterior',
        restoreTitle: 'Restaurar este EDID?',
        restoreFactoryTarget: 'O EDID de fábrica que vem com o NanoKVM.',
        restoreBackupTarget: 'O backup mais recente, aplicado em {{time}}.',
        restoreNotice:
          'Uma restauração é gravada da mesma forma que uma aplicação e tem as mesmas consequências.',
        restored: 'EDID restaurado e verificado.',
        restoreFailed: 'Falha ao restaurar o EDID.',
        mismatchTitle: 'Escrito e relido',
        mismatchWritten: 'Escrito',
        mismatchRead: 'Relido',
        restoreOkBtn: 'Restaurar',
        hardware: 'Hardware detectado: {{hardware}}',
        hardwareUnknown: 'Desconhecido',
        confirmWord: 'APLICAR',
        confirmPrompt: 'Digite {{word}} para habilitar o botão de aplicar.',
        okBtn: 'Aplicar',
        cancelBtn: 'Cancelar'
      },
      presentation: {
        title: 'Apresentação USB',
        loading: 'Carregando...',
        current: 'Apresentação USB atual',
        noProfile: 'Nenhum perfil aplicado',
        linked: 'Funções vinculadas',
        hostState: 'USB do host',
        hostUnbound: 'Controlador não vinculado',
        hdmiState: 'Entrada HDMI',
        hdmiSignal: 'Sinal presente',
        hdmiUnreported: 'Ainda sem relato de captura',
        endpoints: 'Endpoints',
        fifos: 'Slots FIFO',
        pending: 'Alterações pendentes',
        pendingEdits: 'Edições de identidade não salvas',
        pendingProfile: '{{profile}} está selecionado, mas não aplicado',
        pendingNone: 'Nenhuma',
        lastApply: 'Última aplicação',
        applyFailed: 'Falhou em {{profile}} às {{time}}',
        applyClean: 'Nenhuma falha registrada',
        lastKnownGood: 'Último estado bom conhecido',
        rollbackTarget: 'Destino da reversão',
        rollbackNone: 'Nenhum',
        powerCyclePending:
          'O controlador foi tirado do host. Desligue e ligue de novo o computador conectado para recuperar o dispositivo.',
        rollback: 'Reverter',
        rollbackTitle: 'Reverter para {{profile}}?',
        rollbackDesc: 'O gadget é reenumerado; as funções USB caem por um instante.',
        profile: 'Perfil USB',
        builtIn: 'integrado',
        descriptors: 'descritores',
        imported: 'importado',
        clone: 'Clonar',
        cloneTitle: 'Clonar este perfil',
        cloneToEdit:
          'Os perfis integrados continuam somente leitura. Clone este perfil para editar a identidade dele.',
        profileName: 'Nome do perfil',
        profileNameHint: 'Letras minúsculas, números, pontos, sublinhados e hifens.',
        import: 'Importar pacote',
        export: 'Exportar pacote',
        delete: 'Excluir',
        deleteTitle: 'Excluir este perfil?',
        deleteDesc: 'Excluir {{profile}} do NanoKVM.',
        identity: 'Identidade USB',
        preset: 'Identidade predefinida',
        presetPlaceholder: 'Copiar a identidade de um dispositivo conhecido',
        presetHint:
          'Uma predefinição preenche o Vendor ID, o Product ID e os dois campos de nome. Ela não traz descritores.',
        presetSource: 'Identidade conforme registrada em {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'Este Vendor ID pertence a outro fabricante',
        productId: 'Product ID',
        bcdUSB: 'Versão do USB',
        bcdDevice: 'Versão do dispositivo',
        manufacturer: 'Fabricante',
        product: 'Produto',
        serial: 'Número de série',
        configuration: 'Cadeia de configuração',
        hidLayout: 'Dispositivos HID',
        hidRoleKeyboard: 'Teclado',
        hidRoleRelative: 'Mouse (relativo)',
        hidRoleAbsolute: 'Ponteiro (absoluto)',
        hidOff: 'Ausente',
        hidInterface: 'Interface {{index}}',
        hidBootKeyboardShared:
          'O teclado compartilha uma interface, então não oferece mais relatório em protocolo boot. Algumas BIOS e UEFI não vão enxergá-lo.',
        functions: 'Funções',
        descriptorAssets: 'Descritores armazenados: {{count}}',
        endpointUse:
          'IN {{inUse}} em uso, {{inFree}} livres; OUT {{outUse}} em uso, {{outFree}} livres',
        apply: 'Aplicar',
        applyTitle: 'Aplicar este perfil USB?',
        applyDesc: 'O NanoKVM vai apresentar {{profile}} ao computador conectado.',
        reconnect:
          'O teclado, o mouse e as demais funções USB caem por um instante enquanto o gadget é revinculado.',
        applyLinks: 'Vincula: {{functions}}',
        applyRemoves: 'Remove: {{functions}}',
        applyNoHid:
          'Nenhuma função HID sobra depois desta aplicação. O teclado e o mouse param de funcionar.',
        applyRollback: 'Uma aplicação que falhar volta para {{profile}}.',
        recoveryPowerCycle:
          'Nenhum HID sobrevive a esta aplicação, então um host que parar de responder só pode ser recuperado desligando e ligando a energia.',
        recoveryReboot:
          'Uma interface some do dispositivo composto; o host pode precisar de uma reinicialização para vincular o restante de novo.',
        recoveryHdmiReset:
          'Uma função de vídeo é reconstruída, então a cadeia de captura por trás dela é reiniciada.',
        recoveryReconnect: 'O host reenumera o dispositivo; as funções USB caem por um instante.',
        cancel: 'Cancelar',
        noFunctions: 'Nenhuma função vinculada',
        loadFailed: 'Falha ao carregar os perfis de apresentação',
        operationFailed: 'A operação de apresentação falhou'
      },
      passthrough: {
        title: 'Passthrough USB',
        loading: 'Carregando...',
        mode: 'Modo',
        hybrid: 'Híbrido',
        exact: 'Exato',
        hybridDesc: 'Mantém o teclado boot e o mouse relativo, para dispositivos compatíveis.',
        exactDesc: 'Substitui todas as funções USB do NanoKVM pelo dispositivo importado.',
        hybridWarning: 'O modo híbrido mantém o teclado e o mouse relativo',
        hybridWarningDesc:
          'O armazenamento, a rede USB e o ponteiro absoluto se desconectam enquanto a função importada está ativa.',
        hidWarning: 'Iniciar o passthrough abre mão do teclado, do mouse e da mídia virtual',
        hidWarningDesc:
          'O NanoKVM tem um único controlador de dispositivo USB e o proxy precisa dele inteiro, então enquanto uma sessão estiver ativa o host remoto verá o dispositivo repassado em vez do teclado, do mouse e da mídia virtual do NanoKVM. Eles voltam sozinhos assim que a sessão é interrompida. Esta interface web não é afetada, portanto você sempre pode parar a sessão por esta página.',
        hidWarningSafeDesc:
          'O NanoKVM tem apenas um controlador de dispositivo USB e o proxy precisa dele inteiro, então enquanto a sessão roda o host remoto vê o dispositivo redirecionado no lugar do teclado, do mouse e das mídias virtuais do NanoKVM. Eles voltam quando a sessão para.',
        isoLabel: 'Permitir transferências isócronas',
        isoHint:
          'Deixa passar webcams, microfones e outros dispositivos de fluxo. Ninguém mediu o que este hardware aguenta.',
        isoWarning:
          'O fluxo isócrono não é comprovado aqui e pode segurar o teclado e o mouse até você parar a sessão',
        info: {
          title: 'Informações',
          hybrid:
            'O modo híbrido mantém o teclado e o mouse relativo disponíveis. O armazenamento, a rede USB e o ponteiro absoluto se desconectam enquanto o dispositivo importado está ativo.',
          exact:
            'O modo exato substitui todas as funções USB do NanoKVM pelo dispositivo importado. O teclado, o mouse e as mídias virtuais voltam sozinhos quando a sessão para.',
          udc: 'O NanoKVM tem apenas um controlador de dispositivo USB e o proxy precisa dele inteiro, e é por isso que as funções acima somem enquanto durar a sessão.',
          web: 'Esta interface web não é afetada, então você sempre pode parar a sessão por esta página.',
          network:
            'Inicie o passthrough por Ethernet ou Wi-Fi. Iniciá-lo pela rede USB do NanoKVM é recusado, porque essa conexão desapareceria.',
          iso: 'Webcams, microfones e outros dispositivos isócronos são recusados enquanto você não permitir transferências isócronas. Esse caminho funciona, mas nunca foi medido neste hardware: trate a vazão como desconhecida.',
          camera:
            'A câmera e o microfone do navegador, em Dispositivos, continuam sendo a forma comprovada de dar um ao host remoto.'
        },
        session: 'Sessão',
        activeDesc: 'Um dispositivo foi importado e o proxy está com o controlador USB.',
        inactiveDesc:
          'Nenhuma sessão em andamento. O teclado, o mouse e a mídia virtual funcionam normalmente.',
        device: 'Dispositivo',
        busId: 'ID do barramento',
        speed: 'Velocidade',
        exporter: 'Exportador',
        local: 'Importado como',
        localValue: 'Barramento {{bus}}, endereço {{address}}',
        udc: 'Controlador USB',
        pid: 'PID do proxy',
        startedAt: 'Iniciada',
        isoDevice:
          'Este dispositivo transmite por endpoints isócronos, o que nunca foi medido neste hardware',
        exporterLabel: 'Endereço do exportador',
        exporterHint:
          'O host e a porta que o NanoKVM disca. Pelo túnel abaixo isso é {{exporter}}.',
        busIdLabel: 'ID do barramento na sua máquina',
        busIdHint:
          'O busid que o usbip list -l mostra para o dispositivo, por exemplo {{example}}.',
        start: 'Iniciar passthrough',
        stop: 'Parar passthrough',
        startTitle: 'Iniciar o passthrough USB?',
        startDevice: 'O NanoKVM vai importar {{busId}} de {{exporter}}.',
        startHid:
          'O teclado USB, o mouse e a mídia virtual param de funcionar enquanto a sessão durar e voltam sozinhos quando você a interrompe.',
        startIso:
          'Webcams e outros dispositivos isócronos exigem ligar a chave isócrona antes de iniciar.',
        startWeb:
          'Esta interface web continua funcionando, então você pode parar a sessão por esta página a qualquer momento.',
        startNetwork:
          'Use esta página por Ethernet ou Wi-Fi. Iniciar pela rede USB do NanoKVM é recusado porque essa conexão desapareceria.',
        okBtn: 'Iniciar',
        cancelBtn: 'Cancelar',
        instructions: 'Na sua própria máquina',
        instructionsDesc:
          'Por decisão de projeto não há agente cliente para instalar. Execute estes comandos usbip padrão na máquina em que o dispositivo está conectado.',
        copyFailed: 'Falha ao copiar. Copie o comando manualmente.',
        copyInsecure:
          'Esta página não é servida por HTTPS, então o navegador bloqueia a cópia. Copie o comando manualmente ou ative o HTTPS em Configurações, Rede.',
        directNote:
          'Sem túnel, o usbipd precisa estar acessível na sua rede e o endereço do exportador acima precisa apontar para ele. O usbip transporta o dispositivo sem criptografia, então prefira o túnel.',
        steps: {
          modprobe: {
            title: 'Carregar o driver do exportador',
            desc: 'O usbip-host é o que permite ao seu kernel entregar um dispositivo local. Ele não é carregado por padrão.'
          },
          list: {
            title: 'Encontrar o dispositivo',
            desc: 'Lista todos os dispositivos locais com o busid e o par fabricante:produto. Anote o busid do que você quer.'
          },
          bind: {
            title: 'Vincular ao usbip',
            desc: 'Tira o dispositivo do driver normal, então ele deixa de funcionar nesta máquina até você desvinculá-lo.'
          },
          serve: {
            title: 'Publicar o dispositivo',
            desc: 'O usbipd fica em primeiro plano e espera o NanoKVM importar o dispositivo.',
            notice:
              'O usbipd padrão não tem opção de endereço de escuta e escuta em todas as interfaces. Mantenha a porta {{port}} fechada no seu firewall e deixe só o túnel abaixo alcançá-la.'
          },
          tunnel: {
            title: 'Apontar para o NanoKVM',
            desc: 'Um túnel SSH reverso: a porta {{port}} no loopback do próprio NanoKVM passa a ser o usbipd desta máquina. Deixe rodando durante toda a sessão.'
          },
          exporter: {
            title: 'Usar isto como exportador',
            desc: 'Coloque isto no campo do exportador acima, informe o ID do barramento e inicie a sessão.'
          },
          unbind: {
            title: 'Devolver o dispositivo',
            desc: 'Depois que a sessão parar, isto devolve o dispositivo ao driver normal desta máquina.'
          }
        }
      },
      mcp: {
        title: 'Serviço MCP',
        service: 'Controle remoto MCP',
        serviceDesc:
          'Permitir que clientes MCP confiáveis controlem o teclado e o mouse e capturem imagens da tela',
        securityWarning:
          'Qualquer pessoa com esta chave de API pode controlar o host remoto e visualizar sua tela. Use HTTPS e habilite o serviço somente em redes confiáveis.',
        endpoint: 'Endpoint',
        apiKey: 'Chave de API',
        regenerateConfirmTitle: 'Gerar novamente a chave de API MCP?',
        regenerateConfirmDesc: 'A chave atual deixará de funcionar imediatamente.',
        enableConfirmTitle: 'Habilitar o controle MCP externo?',
        enableConfirmDesc:
          'Habilitar o MCP interromperá o PicoClaw e fechará todas as sessões ativas do PicoClaw.',
        failed: 'Falha na operação MCP',
        copyFailed: 'Falha ao copiar. Copie manualmente.',
        copyInsecure:
          'Esta página não é servida por HTTPS, então o navegador bloqueia a cópia. Copie manualmente ou ative o HTTPS em Configurações, Rede.',
        okBtn: 'Confirmar',
        cancelBtn: 'Cancelar'
      },
      about: {
        title: 'Sobre o NanoKVM',
        information: 'Informação',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Versão do Aplicativo',
        applicationTip: 'Versão do aplicativo web NanoKVM',
        image: 'Versão da Imagem',
        imageTip: 'Versão da imagem do sistema NanoKVM',
        deviceKey: 'Chave do Dispositivo',
        community: 'Comunidade',
        hostname: 'Nome do Host',
        hostnameUpdated: 'Nome do host atualizado. Reinicie para aplicar.',
        ipType: {
          Wired: 'Com Fio',
          Wireless: 'Sem Fio',
          Other: 'Outro'
        }
      },
      appearance: {
        title: 'Aparência',
        display: 'Exibição',
        language: 'Idioma',
        languageDesc: 'Selecione o idioma da interface',
        webTitle: 'Título da Web',
        webTitleDesc: 'Personalizar o título da página web',
        favicon: 'Favicon',
        faviconDesc: 'Personalizar o ícone da aba do navegador',
        faviconPlaceholder: 'URL da imagem',
        faviconUpload: 'Enviar',
        faviconReset: 'Redefinir',
        faviconCustom: 'Ícone personalizado',
        faviconBoot: 'Ícone de /boot/logo.ico',
        faviconDefault: 'Ícone padrão',
        faviconOverridesBoot: 'Substituindo /boot/logo.ico',
        faviconErrUrl: 'Informe um endereço de imagem http:// ou https://',
        faviconErrFetch: 'O dispositivo não conseguiu baixar essa imagem',
        faviconErrLarge: 'A imagem é muito grande. O limite é 256 KB',
        faviconErrType: 'Imagem não suportada. Use .ico, .png, .svg, .gif ou .jpg',
        faviconErrSave: 'Falha ao salvar o ícone',
        menuBar: {
          title: 'Barra de Menu',
          mode: 'Modo de exibição',
          modeDesc: 'Exibir barra de menu na tela',
          modeOff: 'Desligado',
          modeAuto: 'Ocultar automaticamente',
          modeAlways: 'Sempre visível',
          keyboardLedStatus: 'Indicadores de bloqueio do teclado',
          keyboardLedStatusDesc:
            'Exibir o estado de Num Lock, Caps Lock e Scroll Lock do computador remoto',
          icons: 'Ícones do submenu',
          iconsDesc: 'Exibir ícones de submenus na barra de menu'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Estado dos bloqueios do teclado remoto',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'Ativado',
        off: 'Desativado',
        unknown: 'Desconhecido'
      },
      device: {
        title: 'Dispositivo',
        oled: {
          title: 'OLED',
          description: 'Desligar tela OLED após',
          0: 'Nunca',
          15: '15 seg',
          30: '30 seg',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 hora'
        },
        ssh: {
          description: 'Habilitar acesso remoto SSH',
          tip: 'Defina uma senha forte antes de habilitar (Conta - Mudar Senha)'
        },
        advanced: 'Configurações Avançadas',
        swap: {
          title: 'Swap',
          disable: 'Desativar',
          description: 'Defina o tamanho do arquivo de swap',
          tip: 'Habilitar esta função pode encurtar a vida útil do seu cartão SD!'
        },
        mouseJiggler: {
          title: 'Movimentador de Mouse',
          description: 'Impedir que o host remoto entre em suspensão',
          disable: 'Desativar',
          absolute: 'Modo Absoluto',
          relative: 'Modo Relativo'
        },
        mdns: {
          description: 'Habilitar serviço de descoberta mDNS',
          tip: 'Desligue se não for necessário'
        },
        hdmi: {
          description: 'Habilitar saída HDMI/monitor',
          idleTimeoutTitle: 'Tempo limite de captura inativa',
          idleTimeoutDescription: 'Parar a captura HDMI após não haver visualizadores ativos por',
          minutes: 'min'
        },
        autostart: {
          title: 'Configurações de scripts de inicialização automática',
          description:
            'Gerencia scripts que são executados automaticamente na inicialização do sistema',
          new: 'Novo',
          deleteConfirm: 'Tem certeza de que deseja excluir este arquivo?',
          yes: 'Sim',
          no: 'Não',
          scriptName: 'Nome do script de inicialização automática',
          scriptContent: 'Conteúdo do script de inicialização automática',
          settings: 'Configurações'
        },
        hidOnly: 'Modo Somente-HID',
        hidOnlyDesc: 'Pare de emular dispositivos virtuais, mantendo apenas o controle básico HID',
        disk: 'Disco Virtual',
        diskDesc: 'Montar U-disk virtual no host remoto',
        rebindNotice:
          'Mudar este interruptor reenumera o dispositivo USB, então o alvo perde por um instante seus dispositivos virtuais e sua rede USB.',
        media: {
          title: 'Slots de câmera e microfone',
          desc: 'Declare os dispositivos de mídia que os navegadores podem ocupar. O orçamento de endpoints é verificado ao aplicar o perfil USB. Salvar reenumera o dispositivo e desconecta os navegadores conectados.',
          cameras: 'Câmeras',
          microphones: 'Microfones',
          name: 'Nome',
          namePlaceholder: 'Exibido no host de destino',
          addCamera: 'Adicionar câmera',
          addMicrophone: 'Adicionar microfone',
          remove: 'Remover',
          cameraDefault: 'Câmera NanoKVM {{index}}',
          microphoneDefault: 'Microfone NanoKVM {{index}}',
          nameRequired: 'Cada slot precisa de um nome.',
          budgetHint:
            'Os seis endpoints USB IN são um limite fixo do hardware. Junte teclado, mouse e ponteiro absoluto em uma única interface HID em Apresentação USB, ou desligue aqui o disco virtual ou, em Rede, o adaptador de rede USB.',
          unsupported:
            'Este kernel não consegue nomear dispositivos de mídia, então os hosts mostram o nome padrão.',
          save: 'Salvar slots',
          disconnect: 'Desconectar',
          disconnectAll: 'Desconectar todas as fontes',
          limit: 'Os slots de câmera e microfone devem somar oito ou menos.',
          failed: 'Não foi possível atualizar os slots de mídia.'
        },
        reboot: 'Reiniciar',
        rebootDesc: 'Tem certeza de que deseja reiniciar o NanoKVM?',
        okBtn: 'Sim',
        cancelBtn: 'Não'
      },
      network: {
        title: 'Rede',
        wifi: {
          title: 'Wi-Fi',
          description: 'Configurar Wi-Fi',
          apMode: 'O modo AP está ativado, conecte-se ao Wi-Fi escaneando o QR code',
          connect: 'Conectar Wi-Fi',
          connectDesc1: 'Digite o SSID da rede e a senha',
          connectDesc2: 'Digite a senha para entrar nesta rede',
          disconnect: 'Tem certeza de que deseja desconectar a rede?',
          failed: 'Falha na conexão, tente novamente.',
          ssid: 'Nome',
          password: 'Senha',
          joinBtn: 'Entrar',
          confirmBtn: 'OK',
          cancelBtn: 'Cancelar'
        },
        tls: {
          description: 'Habilitar protocolo HTTPS',
          tip: 'Atenção: O uso de HTTPS pode aumentar a latência, especialmente com o modo de vídeo MJPEG.'
        },
        usb: {
          title: 'Adaptador de rede USB',
          desc: 'Dá ao computador controlado uma placa de rede por USB',
          type: 'Tipo de adaptador',
          typeDesc: 'NCM para sistemas modernos, RNDIS para Windows antigos'
        },
        bridge: {
          title: 'O adaptador se conecta a',
          lan: 'Sua rede',
          kvmOnly: 'Só o NanoKVM',
          lanDesc:
            'O computador entra na sua rede pela porta Ethernet do NanoKVM, com o seu próprio endereço vindo do roteador.',
          kvmOnlyDesc:
            'O computador recebe o endereço do NanoKVM e alcança o NanoKVM, mas nada além dele.',
          loading: 'Carregando...',
          state: 'Estado',
          states: {
            disabled: 'Só o NanoKVM',
            enabled: 'Sua rede',
            rolledBack: 'Revertido',
            failed: 'Falhou',
            pending: 'Em andamento'
          },
          uplink: 'Uplink',
          ports: 'Portas',
          up: 'ativa',
          down: 'inativa',
          noLink: 'sem link',
          enableTitle: 'Conectar o computador à sua rede?',
          disableTitle: 'Limitar o computador só ao NanoKVM?',
          reconnect:
            'A conexão de gerenciamento cairá e se reconectará brevemente enquanto o endereço é movido.',
          rollback:
            'Se a verificação falhar, a configuração anterior é restaurada automaticamente.',
          enableBtn: 'Entrar na minha rede',
          disableBtn: 'Só o NanoKVM',
          cancelBtn: 'Cancelar',
          interrupted:
            'A conexão foi interrompida durante a aplicação. Verificando novamente o estado atual.',
          pendingNotice:
            'Uma alteração da ponte ainda está em andamento ou foi interrompida antes de terminar.',
          revert: 'Restaurar a configuração anterior',
          rolledBackNotice:
            'A última alteração foi revertida e a configuração anterior foi restaurada.',
          verifyFailed: 'Falha na verificação: {{gates}}',
          gates: {
            address: 'endereço',
            gateway: 'gateway',
            inbound: 'conexão de entrada'
          },
          inboundWeak:
            'A verificação de entrada só passou porque o NanoKVM se conectou a si mesmo. Isso prova que o serviço web está escutando e acessível localmente, não que uma requisição vinda da rede chegue até ele.',
          noCarrier:
            'Sem link em {{port}}. A ponte não tem caminho até a rede enquanto nenhum cabo estiver conectado.',
          loop: 'O roteador também está sendo aprendido em {{port}}, ou seja, essa porta é um segundo caminho para a mesma rede. O spanning tree está desligado, então nada aqui vai quebrar o laço: desconecte um dos dois caminhos.',
          failedNotice:
            'Não foi possível desfazer a última alteração. O NanoKVM pode estar acessível apenas pelo AP Wi-Fi ou por um console serial.'
        },
        dns: {
          title: 'DNS',
          description: 'Configurar servidores DNS para o NanoKVM',
          mode: 'Modo',
          dhcp: 'DHCP',
          manual: 'Manual',
          add: 'Adicionar DNS',
          save: 'Salvar',
          invalid: 'Digite um endereço IP válido',
          noDhcp: 'Nenhum DNS DHCP está disponível no momento',
          saved: 'Configurações de DNS salvas',
          saveFailed: 'Falha ao salvar as configurações de DNS',
          unsaved: 'Alterações não salvas',
          maxServers: 'Máximo de {{count}} servidores DNS permitido',
          dnsServers: 'Servidores DNS',
          dhcpServersDescription: 'Os servidores DNS são obtidos automaticamente via DHCP',
          manualServersDescription: 'Os servidores DNS podem ser editados manualmente',
          networkDetails: 'Detalhes da rede',
          interface: 'Interface',
          ipAddress: 'Endereço IP',
          subnetMask: 'Máscara de sub-rede',
          router: 'Roteador',
          none: 'Nenhum'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'Servidor VNC',
        description:
          'Permite que qualquer cliente VNC veja a tela remota e use o teclado e o mouse, entrando com a sua conta do NanoKVM',
        port: 'Porta',
        portDescription: 'Conecte-se a esta porta no endereço do NanoKVM'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Otimização de memória',
          tip: 'Quando o uso de memória excede o limite, a coleta de lixo é realizada de forma mais agressiva para tentar liberar memória. Recomenda-se definir para 75MB se estiver usando Tailscale. É necessário reiniciar o Tailscale para que a alteração tenha efeito.'
        },
        swap: {
          title: 'Trocar memória',
          tip: 'Se os problemas persistirem após ativar a otimização de memória, tente ativar a memória swap. Isso define o tamanho do arquivo de troca para 256MB por padrão, que pode ser ajustado em "Configurações > Dispositivo".'
        },
        restart: 'Reiniciar Tailscale?',
        stop: 'Parar Tailscale?',
        stopDesc: 'Sair do Tailscale e desabilitar a inicialização automática no boot.',
        loading: 'Carregando...',
        notInstall: 'Tailscale não encontrado! Por favor, instale.',
        install: 'Instalar',
        installing: 'Instalando',
        failed: 'Falha na instalação',
        retry: 'Por favor, atualize e tente novamente. Ou tente instalar manualmente',
        download: 'Baixar o',
        package: 'pacote de instalação',
        unzip: 'e descompacte-o',
        upTailscale: 'Fazer upload do tailscale para o diretório NanoKVM /usr/bin/',
        upTailscaled: 'Fazer upload do tailscaled para o diretório NanoKVM /usr/sbin/',
        refresh: 'Atualizar página atual',
        notRunning: 'Tailscale não está em execução. Por favor, inicie-o para continuar.',
        run: 'Iniciar',
        notLogin:
          'O dispositivo ainda não foi vinculado. Por favor, faça login e vincule este dispositivo à sua conta.',
        urlPeriod: 'Esta URL é válida por 10 minutos',
        login: 'Login',
        loginSuccess: 'Login Bem-sucedido',
        enable: 'Habilitar Tailscale',
        deviceName: 'Nome do Dispositivo',
        deviceIP: 'IP do Dispositivo',
        account: 'Conta',
        logout: 'Sair',
        logoutDesc: 'Tem certeza de que deseja sair?',
        uninstall: 'Desinstalar Tailscale',
        uninstallDesc: 'Tem certeza de que deseja desinstalar Tailscale?',
        okBtn: 'Sim',
        cancelBtn: 'Não'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: 'Carregando...',
        notInstall: 'Não instalado',
        notConfigured: 'Não configurado',
        stopped: 'Parado',
        running: 'Em execução',
        connected: 'Conectado',
        error: 'Erro',
        atBoot: 'inicia na inicialização',
        notAtBoot: 'não inicia na inicialização',
        arguments: 'Argumentos',
        argumentsTip: 'Argumentos de linha de comando passados ao serviço na inicialização.',
        env: 'Variáveis de ambiente',
        envKey: 'Nome',
        envValue: 'Valor',
        envAdd: 'Adicionar variável',
        envRemove: 'Remover',
        configured: 'Configurada',
        save: 'Salvar',
        saved: 'Configuração salva',
        start: 'Iniciar',
        stop: 'Parar',
        restart: 'Reiniciar',
        logs: 'Logs',
        logsEmpty: 'Ainda não há logs',
        refresh: 'Atualizar',
        binary: 'Binário',
        binaryShipped: 'Incluído no firmware',
        binaryCustom: 'Binário personalizado',
        binaryUpload: 'Enviar binário',
        binaryRevert: 'Restaurar binário do firmware',
        binaryRevertDesc: 'Excluir o binário enviado e restaurar o que acompanha o firmware?',
        serverWarning: 'Um servidor sem restrições funciona como proxy aberto',
        noHealthSignal:
          'Este serviço não informa estado de saúde, então o NanoKVM só sabe que o processo está em execução, não se o túnel está conectado.',
        memoryWarning:
          'Executar vários serviços de acesso remoto ao mesmo tempo pode esgotar a memória',
        resources: 'Recursos',
        memory: {
          title: 'Limite de memória',
          description:
            'Limita o heap Go do newt a {{limit}} MiB a partir do próximo reinício. É o limite dele, não o do Tailscale; desligado vale o padrão do Go, com GOGC=50 nos dois casos.',
          noRuntime:
            'O wstunnel é Rust: não há coletor de lixo nem limite de heap a definir, e suas threads de trabalho já acompanham a única CPU do dispositivo.',
          notApplicable: 'Não se aplica'
        },
        swap: {
          title: 'Arquivo de swap',
          description:
            'Adiciona um arquivo de swap de 256 MB no cartão SD. Vale para todo o sistema: o mesmo swap serve o Tailscale, o servidor KVM e tudo o mais no dispositivo.'
        },
        okBtn: 'Sim',
        cancelBtn: 'Não'
      },
      update: {
        title: 'Verificar Atualizações',
        queryFailed: 'Falha ao obter a versão',
        updateFailed: 'Falha na atualização. Por favor, tente novamente.',
        isLatest: 'Você já tem a versão mais recente.',
        rebooting:
          'Instalando o novo kernel e reiniciando. Isso pode levar alguns minutos; não desligue a energia.',
        kernelUpdate:
          'Esta atualização instala o kernel {{version}}. O dispositivo reinicia e volta sozinho ao kernel atual se o novo não iniciar.',
        rolledBack: 'O kernel {{version}} não iniciou e o dispositivo voltou ao kernel anterior.',
        available: 'Uma atualização está disponível. Tem certeza de que deseja atualizar agora?',
        updating: 'Atualização iniciada. Por favor, aguarde...',
        confirm: 'Confirmar',
        cancel: 'Cancelar',
        preview: 'Prévia das Atualizações',
        previewDesc: 'Tenha acesso antecipado a novos recursos e melhorias',
        previewTip:
          'Esteja ciente de que as versões de prévia podem conter bugs ou funcionalidade incompleta!',
        customServer: {
          title: 'Servidor de atualização personalizado',
          desc: 'Verifique e baixe atualizações online de um servidor especificado',
          invalidUrl:
            'Insira um diretório de servidor HTTP ou HTTPS válido, sem parâmetros de consulta, fragmentos ou latest.json.',
          loadFailed: 'Não foi possível carregar a configuração do servidor de atualização.',
          saveFailed: 'Não foi possível salvar a configuração do servidor de atualização.',
          saved: 'Configuração do servidor de atualização salva.',
          save: 'Salvar',
          confirmTitle: 'Usar um servidor de atualização personalizado?',
          confirmDesc:
            'O SHA-512 apenas verifica se o pacote corresponde ao manifesto fornecido por este servidor. Ele não comprova que o pacote seja uma versão oficial do NanoKVM. Um servidor com falha ou mal-intencionado pode inutilizar o dispositivo, causar perda de dados ou comprometer o sistema.',
          confirm: 'Usar mesmo assim',
          previewDisabled:
            'As atualizações de prévia ficam indisponíveis enquanto um servidor de atualização personalizado estiver ativado.'
        },
        offline: {
          kernelNotice:
            'Este pacote contém um kernel. Ele é gravado no slot reserva e o dispositivo reinicia para testá-lo; se não voltar, retorna sozinho ao kernel atual.',
          kernelConfirm: 'Instalar kernel',
          kernelCancel: 'Cancelar',
          title: 'Atualizações off-line',
          desc: 'Atualização através do pacote de instalação local',
          upload: 'Upload',
          checksumPlaceholder: 'Soma de verificação SHA-256 (opcional)',
          invalidChecksum: 'A soma de verificação SHA-256 deve conter 64 caracteres hexadecimais.',
          checksumMismatch: 'A verificação SHA-256 falhou. O pacote pode estar corrompido.',
          invalidName: 'Formato de nome de arquivo inválido. Faça download das versões do GitHub.',
          updateFailed: 'Falha na atualização. Por favor, tente novamente.'
        }
      },
      account: {
        title: 'Conta',
        webAccount: 'Nome da Conta Web',
        role: 'Função',
        roles: {
          admin: 'Administrador',
          user: 'Usuário'
        },
        password: 'Senha',
        updateBtn: 'Alterar',
        logoutBtn: 'Sair',
        logoutDesc: 'Tem certeza de que deseja sair?',
        okBtn: 'Sim',
        cancelBtn: 'Não',
        users: {
          title: 'Usuários',
          create: 'Criar usuário',
          enabled: 'Ativado',
          disabled: 'Desativado',
          deviceOwner: 'Dono do dispositivo',
          resetPassword: 'Redefinir senha',
          delete: 'Excluir',
          deleteConfirm: 'Excluir este usuário e revogar todas as sessões dele?',
          created: 'Usuário criado',
          deleted: 'Usuário excluído',
          passwordUpdated: 'Senha atualizada',
          loadFailed: 'Falha ao carregar os usuários',
          saveFailed: 'Falha ao salvar o usuário',
          deleteFailed: 'Falha ao excluir o usuário'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Assistente',
      empty: 'Abra o painel e inicie uma tarefa para começar.',
      inputPlaceholder: 'Descreva o que você deseja que PicoClaw faça',
      newConversation: 'Nova conversa',
      processing: 'Processando...',
      agent: {
        defaultTitle: 'Assistente geral',
        defaultDescription: 'Ajuda geral sobre bate-papo, pesquisa e espaço de trabalho.',
        kvmTitle: 'Controle remoto',
        kvmDescription: 'Opera o host remoto por meio de NanoKVM.',
        switched: 'Função de agente trocada',
        switchFailed: 'Falha ao mudar de função de agente'
      },
      send: 'Enviar',
      cancel: 'Cancelar',
      status: {
        connecting: 'Conectando ao gateway...',
        connected: 'Sessão PicoClaw conectada',
        disconnected: 'Sessão PicoClaw desconectada',
        stopped: 'Solicitação de parada enviada',
        runtimeStarted: 'Runtime do PicoClaw iniciado',
        runtimeStartFailed: 'Falha ao iniciar o runtime do PicoClaw',
        runtimeStopped: 'Runtime do PicoClaw interrompido',
        runtimeStopFailed: 'Falha ao parar o runtime do PicoClaw',
        controlSwitchedToMCP: 'Controle transferido para o serviço MCP externo'
      },
      connection: {
        runtime: {
          checking: 'Verificando',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime pronto',
          stopped: 'Runtime interrompido',
          blockedByMCP: 'O controle MCP externo está ativo',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime indisponível',
          configError: 'Erro de configuração'
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
          idle: 'Inativo',
          busy: 'Ocupado'
        }
      },
      message: {
        toolAction: 'Ação',
        observation: 'Observação',
        screenshot: 'Captura de tela'
      },
      overlay: {
        locked: 'PicoClaw está controlando o dispositivo. A entrada manual está pausada.'
      },
      control: {
        picoclaw: 'Controle do dispositivo: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Controle do dispositivo: MCP externo',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Controle do dispositivo: desativado',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Conceder controle',
        release: 'Liberar',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'Controle do PicoClaw concedido',
        released: 'Controle do PicoClaw liberado',
        grantFailed: 'Falha ao conceder controle ao PicoClaw',
        releaseFailed: 'Falha ao liberar controle do PicoClaw',
        grantConfirmTitle: 'Alternar controle do dispositivo para PicoClaw?',
        grantConfirmDesc: 'As gravações de dispositivo do MCP externo serão interrompidas.'
      },
      install: {
        install: 'Instalar PicoClaw',
        installing: 'Instalando PicoClaw',
        success: 'PicoClaw instalado com sucesso',
        failed: 'Falha ao instalar PicoClaw',
        uninstalling: 'Desinstalando o runtime...',
        uninstalled: 'Runtime desinstalado com sucesso.',
        uninstallFailed: 'Falha na desinstalação.',
        requiredTitle: 'PicoClaw não está instalado',
        requiredDescription: 'Instale o PicoClaw antes de iniciar o runtime do PicoClaw.',
        progressDescription: 'PicoClaw está sendo baixado e instalado.',
        stages: {
          preparing: 'Preparando',
          downloading: 'Baixando',
          extracting: 'Extraindo',
          verifying: 'Verificando',
          installing: 'Instalando',
          installed: 'Instalado',
          install_timeout: 'Tempo limite esgotado',
          install_failed: 'Falhou'
        }
      },
      model: {
        requiredTitle: 'A configuração do modelo é necessária',
        requiredDescription: 'Configure o modelo PicoClaw antes de usar o chat PicoClaw.',
        docsTitle: 'Guia de configuração',
        docsDesc: 'Modelos e protocolos suportados',
        menuLabel: 'Configurar modelo',
        modelIdentifier: 'Identificador do modelo',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'Chave API',
        apiKeyPlaceholder: 'Insira a chave API do modelo',
        save: 'Salvar',
        saving: 'Salvando',
        saved: 'Configuração do modelo salva',
        saveFailed: 'Falha ao salvar a configuração do modelo',
        invalid: 'Identificador do modelo, API Base URL e chave API são obrigatórios'
      },
      uninstall: {
        menuLabel: 'Desinstalar',
        confirmTitle: 'Desinstalar PicoClaw',
        confirmContent:
          'Tem certeza de que deseja desinstalar PicoClaw? Isso excluirá o executável e todos os arquivos de configuração.',
        confirmOk: 'Desinstalar',
        confirmCancel: 'Cancelar'
      },
      history: {
        title: 'Histórico',
        loading: 'Carregando sessões...',
        emptyTitle: 'Ainda sem histórico',
        emptyDescription: 'As sessões anteriores de PicoClaw aparecerão aqui.',
        loadFailed: 'Falha ao carregar o histórico da sessão',
        deleteFailed: 'Falha ao excluir sessão',
        deleteConfirmTitle: 'Excluir sessão',
        deleteConfirmContent: 'Tem certeza de que deseja excluir "{{title}}"?',
        deleteConfirmOk: 'Excluir',
        deleteConfirmCancel: 'Cancelar',
        messageCount_one: '{{count}} mensagem',
        messageCount_other: '{{count}} mensagens',
        messageCount: '{{count}} mensagens'
      },
      config: {
        startRuntime: 'Iniciar PicoClaw',
        stopRuntime: 'Parar PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Transferir o controle para o PicoClaw?',
        enableConfirmDesc: 'Iniciar o PicoClaw desabilitará o serviço MCP externo.',
        enableConfirmOk: 'Iniciar PicoClaw',
        enableConfirmCancel: 'Cancelar',
        title: 'Iniciar PicoClaw',
        description: 'Inicie o runtime para começar a usar o assistente PicoClaw.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Encontramos um problema',
      refresh: 'Atualizar'
    },
    fullscreen: {
      toggle: 'Alternar Tela Cheia'
    },
    menu: {
      collapse: 'Recolher Menu',
      expand: 'Expandir Menu'
    }
  }
};

export default pt_br;
