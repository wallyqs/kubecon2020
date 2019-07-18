export PS1='\w> '
export PS2='> '

CREDS='-creds .creds'
alias nats-pub='nats-pub $CREDS'
alias nats-sub='nats-sub $CREDS'
alias nats-req='nats-req $CREDS'
alias creds-show='nsc describe jwt -f $CREDS'
